package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"k8s.io/klog"
	"k8s.io/perf-tests/clusterloader2/api"
	"k8s.io/perf-tests/clusterloader2/pkg/config"
	"k8s.io/perf-tests/clusterloader2/pkg/flags"
	"k8s.io/perf-tests/clusterloader2/pkg/framework"
	"k8s.io/perf-tests/clusterloader2/pkg/provider"
	"k8s.io/perf-tests/clusterloader2/pkg/test"

	_ "net/http/pprof"

	_ "k8s.io/perf-tests/clusterloader2/pkg/measurement/common"
	_ "k8s.io/perf-tests/clusterloader2/pkg/measurement/common/bundle"
	_ "k8s.io/perf-tests/clusterloader2/pkg/measurement/common/dns"
	_ "k8s.io/perf-tests/clusterloader2/pkg/measurement/common/network"
	_ "k8s.io/perf-tests/clusterloader2/pkg/measurement/common/probes"
	_ "k8s.io/perf-tests/clusterloader2/pkg/measurement/common/slos"
	_ "sigs.k8s.io/kueue/perf-tests/clusterloader2/measurement"
)

var (
	testConfigPaths     []string
	testSuiteConfigPath string
	providerName        string = "local"
	port                int
	kubeconfigPath      string
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func DefaultClusterLoaderConfig() config.ClusterLoaderConfig {
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	if kubeconfigPath == "" {
		kubeconfigPath = path.Join(home, "/.kube/config")
	}
	if _, err := os.Stat(kubeconfigPath); errors.Is(err, os.ErrNotExist) {
		klog.Errorf("Kubeconfig %s does not exist", kubeconfigPath)
	}
	result := config.ClusterLoaderConfig{
		ReportDir: getenv("REPORT_DIR", path.Join(cwd, "reports")),
		ClusterConfig: config.ClusterConfig{
			KubeConfigPath: kubeconfigPath,
			Provider:       provider.NewLocalProvider(nil),
			RunFromCluster: false,
		},
	}
	if result.ClusterConfig.K8SClientsNumber == 0 {
		// HACK to avoid division by zero later
		result.ClusterConfig.K8SClientsNumber = 1
	}
	return result
}

func testID(ts *api.TestScenario) string {
	if ts.Identifier != "" {
		return fmt.Sprintf("%s(%s)", ts.Identifier, ts.ConfigPath)
	}
	return ts.ConfigPath
}

func printTestResult(name, status, errors string) {
	logf := klog.V(0).Infof
	if errors != "" {
		logf = klog.Errorf
	}
	logf("Test: %v", name)
	logf("  Status: %v", status)
	if errors != "" {
		logf("  Errors: %v", errors)
	}
}

func RunTest(ctx test.Context) {
	testID := testID(ctx.GetTestScenario())
	klog.V(0).Infof("Running test: %v", testID)
	testStart := time.Now()
	errList := test.RunTest(ctx)
	if errList.IsEmpty() {
		printTestResult(testID, "Success", "")
	} else {
		printTestResult(testID, "Fail", errList.String())
	}
	testConfigPath := ctx.GetTestScenario().ConfigPath
	ctx.GetTestReporter().ReportTestFinish(time.Since(testStart), testConfigPath, errList)
}

func GetTestScenariosFromConfigs(testPaths ...string) []api.TestScenario {
	cwd, _ := os.Getwd()
	var results []api.TestScenario
	if len(testPaths) == 0 {
		testPaths = []string{path.Join(cwd, "scenarios")}
	}
	for _, testPath := range testPaths {
		if testPath == "" {
			continue
		}

		info, err := os.Stat(testPath)
		if errors.Is(err, os.ErrNotExist) {
			klog.Warningf("Path %s does not exist, ignoring", testPath)
			continue
		}
		if info.IsDir() {
			pattern := path.Join(testPath, "config**")
			files, _ := filepath.Glob(pattern)
			for _, file := range files {
				dirFileInfo, _ := os.Stat(file)
				if !dirFileInfo.IsDir() {
					ext := filepath.Ext(file)
					if ext == ".yaml" || ext == ".yml" || ext == ".json" {
						results = append(results, api.TestScenario{ConfigPath: file})
					}
				}
			}
		} else {
			results = append(results, api.TestScenario{ConfigPath: testPath})
		}
	}
	return results
}

func GetTestScenariosFromSuite(paths ...string) []api.TestScenario {
	path := os.Getenv("TEST_SUITE")
	if path != "" {
		paths = append(paths, path)
	}
	var results []api.TestScenario
	for _, path := range paths {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			klog.Warningf("Test suite %s does not exist", path)
			return []api.TestScenario{}
		}
		testSuite, _ := config.LoadTestSuite(path)
		results = append(results, []api.TestScenario(testSuite)...)
	}
	return results
}

func GetPrometheusFramework() *framework.Framework {
	// Not implemented yet
	return nil
}

func init() {
	flags.IntEnvVar(&port, "port", "PORT", 8000, "Port to be used by http server with pprof.")
	flags.StringEnvVar(&providerName, "provider", "PROVIDER", "", "Cluster provider name")
	flags.StringSliceEnvVar(&testConfigPaths, "testconfig", "TEST", []string{}, "Paths to the test config files")
	flags.StringEnvVar(&kubeconfigPath, "kubeconfig", "KUBECONFIG", "", "Path to the kubeconfig file (if not empty, --run-from-cluster must be false)")
	flags.StringVar(&testSuiteConfigPath, "testsuite", "", "Path to the test suite config file")
}

func main() {
	defer klog.Flush()
	infof := klog.V(0).Infof

	if err := flags.Parse(); err != nil {
		klog.Exitf("Flag parse failed: %v", err)
	}

	conf := DefaultClusterLoaderConfig()
	prometheusFramework := GetPrometheusFramework()

	framework, _ := framework.NewFramework(&conf.ClusterConfig, 1)

	// mclient, _ := framework.NewMultiClientSet(config.ClusterConfig.KubeConfigPath, 1)
	// Start http server with pprof.
	go func() {
		infof("Listening on %d", port)
		err := http.ListenAndServe(fmt.Sprintf("localhost:%d", port), nil)
		klog.Errorf("http server unexpectedly ended: %v", err)
	}()

	testReporter := test.CreateSimpleReporter(path.Join(conf.ReportDir, "junit.xml"), "ClusterLoaderV2")
	testReporter.BeginTestSuite()

	var testScenarios []api.TestScenario
	if testSuiteConfigPath != "" {
		testScenarios = GetTestScenariosFromSuite(testSuiteConfigPath)
	} else {
		testScenarios = GetTestScenariosFromConfigs(testConfigPaths...)
	}

	var contexts []test.Context
	for i := range testScenarios {
		testId := testID(&testScenarios[i])
		infof("Compiling test scenario %s", testId)
		ctx, errList := test.CreateTestContext(framework, prometheusFramework, &conf, testReporter, &testScenarios[i])
		if !errList.IsEmpty() {
			klog.Warningf("Test context creation failed: %s", errList.String())
			continue
		}
		testConfig, errList := test.CompileTestConfig(ctx)
		if !errList.IsEmpty() {
			klog.Warningf("Test compilation failed: %s", errList.String())
			continue
		}
		ctx.SetTestConfig(testConfig)
		contexts = append(contexts, ctx)
	}
	if len(contexts) == 0 {
		klog.Exitf("No tests to run")
	}

	os.MkdirAll(conf.ReportDir, 0755)
	for i := range contexts {
		RunTest(contexts[i])
	}

	testReporter.EndTestSuite()
}
