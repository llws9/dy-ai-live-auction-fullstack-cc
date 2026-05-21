package apitest

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Env is the concise public name used by human-written tests.
type Env = EnvConfig

// TestContext is the flat, Tesla-Go-like entry point for generated cases.
// It keeps gateway/env/log state out of the test body while still allowing
// tests to read like: req -> Call -> Assert.
type TestContext struct {
	t      *testing.T
	suite  *Suite
	caseID string
	vars   map[string]any
}

// HTTPRequest describes one HTTP call through paas-gw.
type HTTPRequest struct {
	Name       string
	Method     string
	Path       string
	Headers    map[string]string
	Params     map[string]string
	Body       any
	Extract    map[string]string
	RpcContext map[string]string
}

// RPCRequest describes one RPC call through paas-gw.
type RPCRequest struct {
	Name       string
	Method     string
	Headers    map[string]string
	RpcContext map[string]string
	Body       any
	Extract    map[string]string
}

// Response is the public result returned by CallHTTP/CallRPC.
type Response struct {
	StepResult
}

// EnvFromFile loads APITEST_ENV and skips with a runnable hint when it is absent.
func EnvFromFile(t *testing.T) *EnvConfig {
	t.Helper()
	envFile := os.Getenv("APITEST_ENV")
	if envFile == "" {
		t.Skip("APITEST_ENV not set; set it to FEATURE_DIR/test/.env, e.g. `export APITEST_ENV=$(pwd)/specs/<feature>/test/.env`. Run user_jwt in the workflow or export APITEST_TOKEN=<jwt> for paas-gw.")
	}
	cfg, err := LoadEnv(envFile)
	if err != nil {
		t.Fatalf("apitest: load env: %v", err)
	}
	return cfg
}

// NewContext creates the flat API context. Token and log dir defaults are read
// from the same environment variables used by the older Suite template.
func NewContext(t *testing.T, env *EnvConfig) *TestContext {
	t.Helper()
	env = mergeEnvFromFile(t, env)
	token := envToken(env)
	if token == "" {
		t.Skip("paas-gw JWT not set; run user_jwt in the workflow or export APITEST_TOKEN=<jwt>")
	}

	logDir := os.Getenv("APITEST_LOG_DIR")
	if logDir == "" {
		if envFile := os.Getenv("APITEST_ENV"); envFile != "" {
			logDir = filepath.Join(filepath.Dir(envFile), "api_test_logs")
		} else {
			logDir = "api_test_logs"
		}
	}
	_ = os.MkdirAll(logDir, 0o755)

	suite := New(t).
		WithEnv(env).
		WithJWTToken(token).
		WithLogDir(logDir)
	return &TestContext{
		t:      t,
		suite:  suite,
		caseID: safeCaseID(t.Name()),
		vars:   map[string]any{},
	}
}

func mergeEnvFromFile(t *testing.T, env *EnvConfig) *EnvConfig {
	t.Helper()
	if env == nil {
		env = &EnvConfig{}
	}
	envFile := os.Getenv("APITEST_ENV")
	if envFile == "" {
		return env
	}
	fileEnv, err := LoadEnv(envFile)
	if err != nil {
		t.Fatalf("apitest: load env: %v", err)
	}
	merged := *fileEnv
	if env.PSM != "" {
		merged.PSM = env.PSM
	}
	if env.Host != "" {
		merged.Host = env.Host
	}
	if env.Env != "" {
		merged.Env = env.Env
	}
	if env.Branch != "" {
		merged.Branch = env.Branch
	}
	if env.Zone != "" {
		merged.Zone = env.Zone
	}
	if env.IDC != "" {
		merged.IDC = env.IDC
	}
	if env.Cluster != "" {
		merged.Cluster = env.Cluster
	}
	if env.TestAccount != nil {
		merged.TestAccount = env.TestAccount
	}
	return &merged
}

// WithCaseID overrides the log case id used by CallHTTP/CallRPC.
func (c *TestContext) WithCaseID(id string) *TestContext {
	c.caseID = id
	return c
}

// WithVars adds variables available to later request bodies/params via
// ${{var}} or ${var} placeholders.
func (c *TestContext) WithVars(vars map[string]any) *TestContext {
	for k, v := range vars {
		c.vars[k] = v
	}
	return c
}

// DeferCleanup registers cleanup tied to the active test case.
func (c *TestContext) DeferCleanup(fn func()) {
	c.t.Cleanup(fn)
}

// CallHTTP executes one HTTP request and returns its response.
func CallHTTP(ctx *TestContext, req HTTPRequest) Response {
	ctx.t.Helper()
	name := req.Name
	if name == "" {
		name = strings.TrimSpace(req.Method + " " + req.Path)
	}
	res := ctx.suite.RunStep(ctx.caseID, Step{
		Name:       name,
		Type:       "HTTP",
		API:        req.Path,
		Method:     req.Method,
		Headers:    req.Headers,
		Params:     req.Params,
		RpcContext: req.RpcContext,
		Body:       req.Body,
		Extract:    req.Extract,
	}, ctx.vars)
	mergeExtracted(ctx.vars, res.Extracted)
	return Response{StepResult: res}
}

// CallRPC executes one RPC request and returns its response.
func CallRPC(ctx *TestContext, req RPCRequest) Response {
	ctx.t.Helper()
	name := req.Name
	if name == "" {
		name = req.Method
	}
	res := ctx.suite.RunStep(ctx.caseID, Step{
		Name:       name,
		Type:       "RPC",
		API:        req.Method,
		Headers:    req.Headers,
		RpcContext: req.RpcContext,
		Body:       req.Body,
		Extract:    req.Extract,
	}, ctx.vars)
	mergeExtracted(ctx.vars, res.Extracted)
	return Response{StepResult: res}
}

// Assert evaluates response expressions using the same grammar as Step.Asserts.
func Assert(t *testing.T, resp Response, expressions ...string) {
	t.Helper()
	asserts := evaluateAll(expressions, resp.StatusCode, resp.Body)
	for _, a := range asserts {
		if a.Passed {
			continue
		}
		if a.Err != nil {
			t.Errorf("assert %q failed: %v", a.Expression, a.Err)
		} else {
			t.Errorf("assert %q failed (actual=%v)", a.Expression, a.Actual)
		}
	}
}

// Value extracts a JSONPath value from the response body.
func (r Response) Value(path string) any {
	v, _ := jsonPathExtract(r.Body, path)
	return v
}

// ExtractString extracts a JSONPath value and stringifies it for ordinary Go
// variable passing between calls.
func (r Response) ExtractString(path string) string {
	v := r.Value(path)
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// ExtractInt64 extracts a JSONPath value as int64 when possible.
func (r Response) ExtractInt64(path string) int64 {
	switch v := r.Value(path).(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// T returns the active testing.T for suite-style tests.
func (s *Suite) T() *testing.T {
	return s.t
}

// Assert lets suite methods use s.Assert(resp, "...") without passing t around.
func (s *Suite) Assert(resp Response, expressions ...string) {
	Assert(s.t, resp, expressions...)
}

// DeferCleanup registers cleanup tied to the active suite case.
func (s *Suite) DeferCleanup(fn func()) {
	s.t.Cleanup(fn)
}

// RunSuite runs Tesla-Go-style suite tests. It recognizes optional lifecycle
// methods named SuiteSetup, SuiteTeardown, CaseSetup and CaseTeardown.
func RunSuite(t *testing.T, suite any) {
	t.Helper()
	setEmbeddedSuite(t, suite)
	callNoArg(suite, "SuiteSetup")
	defer callNoArg(suite, "SuiteTeardown")

	v := reflect.ValueOf(suite)
	typ := v.Type()
	suiteName := strings.TrimPrefix(typ.Elem().Name(), "*")
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if !strings.HasPrefix(method.Name, "Test") || method.Type.NumIn() != 1 {
			continue
		}
		testName := method.Name
		t.Run(testName, func(st *testing.T) {
			setEmbeddedSuite(st, suite)
			callLifecycle(suite, "CaseSetup", suiteName, testName)
			defer callLifecycle(suite, "CaseTeardown", suiteName, testName)
			method.Func.Call([]reflect.Value{v})
		})
	}
}

func setEmbeddedSuite(t *testing.T, suite any) {
	v := reflect.ValueOf(suite)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		t.Fatalf("apitest: RunSuite requires pointer to struct, got %T", suite)
	}
	field := v.Elem().FieldByName("Suite")
	if !field.IsValid() || !field.CanSet() {
		t.Fatalf("apitest: suite %T must embed apitest.Suite", suite)
	}
	next := *New(t)
	if existing, ok := field.Addr().Interface().(*Suite); ok {
		next.env = existing.env
		next.client = existing.client
		next.logDir = existing.logDir
		next.globals = cloneVarMap(existing.globals)
	}
	field.Set(reflect.ValueOf(next))
}

func callNoArg(receiver any, name string) {
	m := reflect.ValueOf(receiver).MethodByName(name)
	if m.IsValid() && m.Type().NumIn() == 0 {
		m.Call(nil)
	}
}

func callLifecycle(receiver any, name, suiteName, testName string) {
	m := reflect.ValueOf(receiver).MethodByName(name)
	if m.IsValid() && m.Type().NumIn() == 2 {
		m.Call([]reflect.Value{reflect.ValueOf(suiteName), reflect.ValueOf(testName)})
	}
}

func mergeExtracted(vars map[string]any, extracted map[string]any) {
	for k, v := range extracted {
		vars[k] = v
	}
}

func safeCaseID(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}
