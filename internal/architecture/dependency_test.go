package architecture_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const modulePath = "github.com/maguowei/gotobeta"

type goPackage struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

// skippedPrefixes 列出不参与分层断言的包：
// - internal/app 是组合根，允许装配所有层；
// - internal/ent 是生成代码；
// - architecture/testutil/integration 是测试基础设施。
var skippedPrefixes = []string{
	modulePath + "/internal/app",
	modulePath + "/internal/ent",
	modulePath + "/internal/architecture",
	modulePath + "/internal/testutil",
	modulePath + "/internal/integration",
}

// sdkGateways 声明第三方 SDK 的唯一归口：SDK 只能被白名单内的封装包直接 import，
// 防止厂商依赖扩散。只约束生产 import（Imports），测试文件可为构造夹具直接使用 SDK。
// 未启用对应能力的项目不存在相应 import，规则天然安全。
var sdkGateways = []struct {
	sdkPrefix string
	allowed   []string
	reason    string
}{
	{"github.com/IBM/sarama", []string{modulePath + "/internal/infra/eventbus"}, "Kafka SDK 只能通过 infra/eventbus 封装使用"},
	{"github.com/redis/go-redis", []string{modulePath + "/internal/infra/cache"}, "Redis SDK 只能通过 infra/cache 封装使用"},
	{"go.opentelemetry.io/otel/sdk", []string{modulePath + "/internal/pkg/trace"}, "OTel SDK 初始化只能在 pkg/trace（API 包 otel/otel/trace 不受限）"},
	{"go.opentelemetry.io/otel/exporters", []string{modulePath + "/internal/pkg/trace"}, "OTel exporter 只能在 pkg/trace 配置"},
	{"github.com/getsentry/sentry-go", []string{modulePath + "/internal/infra/sentry", modulePath + "/internal/pkg/logger", modulePath + "/internal/pkg/httpx/middleware"}, "Sentry SDK 只能通过 infra/sentry、pkg/logger 与 Recovery/Sentry 中间件使用"},
	{"github.com/golang-jwt/jwt", []string{modulePath + "/internal/pkg/auth"}, "JWT SDK 只能通过 pkg/auth 封装使用"},
	{"github.com/minio/minio-go", []string{modulePath + "/internal/infra/objstore"}, "对象存储 SDK 只能通过 infra/objstore 封装使用"},
}

// cmdAllowedInternalPrefixes 是 cmd/** 允许 import 的本模块包白名单：
// main 必须保持极薄，只做信号处理、bootstrap 与组合根调用。
// 只约束生产 import；main_test 为 stub 函数签名引用 typed config 是合法的。
var cmdAllowedInternalPrefixes = []string{
	modulePath + "/internal/app",
}

func TestDDDDependencyBoundaries(t *testing.T) {
	packages := listProjectPackages(t)

	for _, pkg := range packages {
		if isSkipped(pkg.ImportPath) {
			continue
		}
		imports := allImports(pkg)

		// 全局规则一：生产代码禁止依赖测试基础设施（只查 Imports，不含测试 import）。
		assertNoForbiddenImports(t, pkg, pkg.Imports, "production code must not depend on test infrastructure", forbiddenImportPrefixes{
			modulePath + "/internal/testutil": "testutil is test-only infrastructure",
		})

		// 全局规则二：第三方 SDK 唯一归口（只查 Imports）。
		assertSDKGateways(t, pkg)

		// 全局规则三：跨模块边界——模块之间只能通过 internal/pkg 共享契约或领域事件协作，
		// 不得直接 import 另一个模块的任何包。
		assertNoCrossModuleImports(t, pkg, imports)

		switch {
		case strings.Contains(pkg.ImportPath, "/internal/modules/") && strings.Contains(pkg.ImportPath, "/domain/"):
			assertNoCrossAggregateImports(t, pkg, imports)
			assertNoForbiddenImports(t, pkg, imports, "domain packages must not import outer layers", forbiddenImportPrefixes{
				modulePath + "/internal/infra":                 "domain must stay independent from global infrastructure",
				modulePath + "/internal/app":                   "domain must not depend on process composition roots",
				modulePath + "/internal/ent":                   "domain must not depend on generated Ent packages",
				modulePath + "/internal/modules/*/application": "domain must not depend on application layer",
				modulePath + "/internal/modules/*/infra":       "domain must not depend on infrastructure layer",
				modulePath + "/internal/modules/*/adapter":     "domain must not depend on adapter layer",
				"github.com/gin-gonic/gin":                     "domain must not depend on HTTP framework",
				"github.com/spf13/viper":                       "domain must not depend on config framework",
				"entgo.io/ent":                                 "domain must not depend on Ent",
				"database/sql":                                 "domain must not depend on database/sql",
				"net/http":                                     "domain must not depend on net/http",
				"log/slog":                                     "domain must not depend on slog",
				"github.com/go-resty/resty/v2":                 "domain must not depend on Resty",
			})
		case strings.Contains(pkg.ImportPath, "/internal/modules/") && strings.Contains(pkg.ImportPath, "/application/"):
			forbidden := forbiddenImportPrefixes{
				modulePath + "/internal/infra":             "application must use injected ports instead of global infrastructure",
				modulePath + "/internal/app":               "application must not depend on process composition roots",
				modulePath + "/internal/ent":               "application must not depend on generated Ent packages",
				modulePath + "/internal/modules/*/infra":   "application must not depend on infrastructure layer",
				modulePath + "/internal/modules/*/adapter": "application must not depend on adapter layer",
				"github.com/gin-gonic/gin":                 "application must not depend on HTTP framework",
				"github.com/spf13/viper":                   "application must not depend on config framework",
				"entgo.io/ent":                             "application must not depend on Ent",
				"net/http":                                 "application must not depend on net/http",
				"github.com/go-resty/resty/v2":             "application must use injected ports instead of Resty",
			}
			// CQRS 约束：查询是只读路径，不得引用事件发布契约。
			if strings.Contains(pkg.ImportPath, "/application/query") {
				forbidden[modulePath+"/internal/pkg/event"] = "queries are read-only and must not publish domain events"
			}
			assertNoForbiddenImports(t, pkg, imports, "application packages must not import infrastructure or adapter packages", forbidden)
		case strings.Contains(pkg.ImportPath, "/internal/modules/") && strings.Contains(pkg.ImportPath, "/adapter/"):
			forbidden := forbiddenImportPrefixes{
				modulePath + "/internal/infra":           "adapter must use application ports instead of global infrastructure",
				modulePath + "/internal/app":             "adapter must not depend on process composition roots",
				modulePath + "/internal/modules/*/infra": "adapter must not depend on infrastructure layer",
				modulePath + "/internal/ent":             "adapter must not depend on generated Ent packages",
				"entgo.io/ent":                           "adapter must not depend on Ent",
				"database/sql":                           "adapter must not depend on database/sql",
			}
			// HTTP 契约只能从 application 的 command/query/result 映射而来，禁止泄漏领域对象。
			if strings.Contains(pkg.ImportPath, "/adapter/http/request") || strings.Contains(pkg.ImportPath, "/adapter/http/response") {
				forbidden[modulePath+"/internal/modules/*/domain"] = "HTTP request/response contracts must map from application command/query/result, not domain objects"
			}
			// handler 只接触应用层出入参，不直接操作领域实体。
			if strings.Contains(pkg.ImportPath, "/adapter/http/handler") {
				forbidden[modulePath+"/internal/modules/*/domain"] = "handlers must work with application commands/queries/results instead of domain objects"
			}
			assertNoForbiddenImports(t, pkg, imports, "adapter packages must not import infrastructure packages", forbidden)
		case strings.Contains(pkg.ImportPath, "/internal/modules/") && strings.Contains(pkg.ImportPath, "/infra/"):
			assertNoForbiddenImports(t, pkg, imports, "infrastructure packages must not import application or adapter packages", forbiddenImportPrefixes{
				modulePath + "/internal/app":                   "infrastructure must not depend on process composition roots",
				modulePath + "/internal/modules/*/application": "infrastructure must not depend on application layer",
				modulePath + "/internal/modules/*/adapter":     "infrastructure must not depend on adapter layer",
				"github.com/gin-gonic/gin":                     "infrastructure must not depend on HTTP framework",
			})
		case strings.HasPrefix(pkg.ImportPath, modulePath+"/internal/pkg/"):
			forbidden := forbiddenImportPrefixes{
				modulePath + "/internal/app":     "shared kernel must not depend on process composition roots",
				modulePath + "/internal/modules": "shared kernel must not depend on business modules",
				"entgo.io/ent":                   "shared kernel must not depend on Ent",
			}
			if !strings.HasPrefix(pkg.ImportPath, modulePath+"/internal/pkg/httpx") {
				forbidden["github.com/gin-gonic/gin"] = "only internal/pkg/httpx may depend on the HTTP framework"
			}
			forbidden[modulePath+"/internal/infra"] = "shared kernel must stay independent from global infrastructure"
			assertNoForbiddenImports(t, pkg, imports, "internal/pkg packages must stay framework-agnostic", forbidden)
		case strings.HasPrefix(pkg.ImportPath, modulePath+"/internal/infra"):
			assertNoForbiddenImports(t, pkg, imports, "global infrastructure must not depend on upper layers", forbiddenImportPrefixes{
				modulePath + "/internal/app":     "infrastructure must not depend on process composition roots",
				modulePath + "/internal/modules": "global infrastructure must stay agnostic of business modules",
				"github.com/gin-gonic/gin":       "infrastructure must not depend on HTTP framework",
			})
		case strings.HasPrefix(pkg.ImportPath, modulePath+"/pkg/"):
			assertNoForbiddenImports(t, pkg, imports, "public pkg packages must stay free of internal dependencies", forbiddenImportPrefixes{
				modulePath + "/internal": "pkg/ is published code and must not import internal/",
			})
		case strings.HasPrefix(pkg.ImportPath, modulePath+"/cmd/"):
			assertOnlyAllowedInternalImports(t, pkg, "cmd packages must stay thin composition entrypoints", cmdAllowedInternalPrefixes)
		}
	}
}

func listProjectPackages(t *testing.T) []goPackage {
	t.Helper()

	root := projectRoot(t)
	cmd := exec.Command("go", "list", "-json", "./internal/...", "./pkg/...", "./cmd/...")
	cmd.Dir = root

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list project packages: %v\n%s", err, string(exitErr.Stderr))
		}
		t.Fatalf("go list project packages: %v", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	packages := make([]goPackage, 0)
	for decoder.More() {
		var pkg goPackage
		if err := decoder.Decode(&pkg); err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		packages = append(packages, pkg)
	}
	return packages
}

func isSkipped(importPath string) bool {
	for _, prefix := range skippedPrefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func allImports(pkg goPackage) []string {
	imports := make([]string, 0, len(pkg.Imports)+len(pkg.TestImports)+len(pkg.XTestImports))
	imports = append(imports, pkg.Imports...)
	imports = append(imports, pkg.TestImports...)
	imports = append(imports, pkg.XTestImports...)
	return imports
}

// moduleNameOf 提取 import 路径所属的业务模块名；非模块包返回空串。
func moduleNameOf(importPath string) string {
	const marker = "/internal/modules/"
	_, rest, ok := strings.Cut(importPath, marker)
	if !ok {
		return ""
	}
	if before, _, ok := strings.Cut(rest, "/"); ok {
		return before
	}
	return rest
}

// assertNoCrossModuleImports 断言模块包不会直接 import 另一个模块的包。
func assertNoCrossModuleImports(t *testing.T, pkg goPackage, imports []string) {
	t.Helper()

	own := moduleNameOf(pkg.ImportPath)
	if own == "" {
		return
	}
	for _, imported := range imports {
		if !strings.HasPrefix(imported, modulePath+"/internal/modules/") {
			continue
		}
		if other := moduleNameOf(imported); other != "" && other != own {
			t.Fatalf("cross-module dependency: package %s imports %s (modules must collaborate via internal/pkg contracts or domain events)", pkg.ImportPath, imported)
		}
	}
}

// assertSDKGateways 断言第三方 SDK 只被声明的归口包直接 import（只查生产 Imports）。
func assertSDKGateways(t *testing.T, pkg goPackage) {
	t.Helper()

	for _, gw := range sdkGateways {
		allowed := false
		for _, prefix := range gw.allowed {
			if pkg.ImportPath == prefix || strings.HasPrefix(pkg.ImportPath, prefix+"/") {
				allowed = true
				break
			}
		}
		if allowed {
			continue
		}
		for _, imported := range pkg.Imports {
			if imported == gw.sdkPrefix || strings.HasPrefix(imported, gw.sdkPrefix+"/") {
				t.Fatalf("SDK gateway violation: package %s imports %s (%s)", pkg.ImportPath, imported, gw.reason)
			}
		}
	}
}

// assertOnlyAllowedInternalImports 断言包的生产 import 中，本模块路径只出现在白名单前缀内。
func assertOnlyAllowedInternalImports(t *testing.T, pkg goPackage, message string, allowedPrefixes []string) {
	t.Helper()

	for _, imported := range pkg.Imports {
		if !strings.HasPrefix(imported, modulePath+"/") {
			continue
		}
		allowed := false
		for _, prefix := range allowedPrefixes {
			if imported == prefix || strings.HasPrefix(imported, prefix+"/") {
				allowed = true
				break
			}
		}
		if !allowed {
			t.Fatalf("%s: package %s imports %s (allowed prefixes: %s)", message, pkg.ImportPath, imported, strings.Join(allowedPrefixes, ", "))
		}
	}
}

type forbiddenImportPrefixes map[string]string

func assertNoForbiddenImports(t *testing.T, pkg goPackage, imports []string, message string, forbidden forbiddenImportPrefixes) {
	t.Helper()

	for _, imported := range imports {
		for prefix, reason := range forbidden {
			if importMatches(prefix, imported) {
				t.Fatalf("%s: package %s imports %s (%s)", message, pkg.ImportPath, imported, reason)
			}
		}
	}
}

func importMatches(pattern string, imported string) bool {
	if strings.Contains(pattern, "*") {
		before, after, ok := strings.Cut(pattern, "*")
		if !ok || strings.Contains(after, "*") {
			return false
		}
		return strings.HasPrefix(imported, before) && strings.Contains(imported[len(before):], after)
	}
	return imported == pattern || strings.HasPrefix(imported, pattern+"/")
}

// assertNoCrossAggregateImports 断言同一模块内的聚合包不互相 import；
// 跨聚合协调应在 application 层完成。
func assertNoCrossAggregateImports(t *testing.T, pkg goPackage, imports []string) {
	t.Helper()

	const domainSeg = "/domain/"
	modIdx := strings.Index(pkg.ImportPath, "/internal/modules/")
	if modIdx < 0 {
		return
	}
	domainIdx := strings.Index(pkg.ImportPath[modIdx:], domainSeg)
	if domainIdx < 0 {
		return
	}
	domainBase := pkg.ImportPath[:modIdx+domainIdx+len(domainSeg)]
	rest := pkg.ImportPath[len(domainBase):]
	ownAggregate, _, _ := strings.Cut(rest, "/")
	if ownAggregate == "" {
		return
	}

	for _, imported := range imports {
		if !strings.HasPrefix(imported, domainBase) {
			continue
		}
		importedRest := imported[len(domainBase):]
		otherAggregate, _, _ := strings.Cut(importedRest, "/")
		if otherAggregate != "" && otherAggregate != ownAggregate {
			t.Fatalf("cross-aggregate dependency: package %s imports %s (aggregate packages must not import each other; cross-aggregate coordination belongs in the application layer)", pkg.ImportPath, imported)
		}
	}
}
