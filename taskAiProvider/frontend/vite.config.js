import path from "path";
import fs from "node:fs";
import { execSync } from "node:child_process";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { fileURLToPath, URL } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/** 与 monorepo conf/（scripts/conf-read.py snapshot-json）一致 */
function loadMergedPortConfig() {
  try {
    const monorepoRoot = path.resolve(__dirname, "../../..");
    const script = path.join(monorepoRoot, "runAll", "scripts", "conf-read.py");
    const raw = execSync(`python3 "${script}" snapshot-json`, {
      cwd: monorepoRoot,
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "pipe"],
    });
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

function readAiProviderAllowedHosts() {
  const j = loadMergedPortConfig();
  const raw = (j.aiProvider || {}).allowedHost;
  if (typeof raw === "string" && raw.trim()) {
    try {
      const u = new URL(raw.trim());
      return u.hostname ? [u.hostname] : [];
    } catch {
      return [];
    }
  }
  return [];
}

function readAiProviderApiTarget() {
  const fromEnv =
    process.env.AI_PROVIDER_DEV_PORT || process.env.AI_PROVIDER_PORT;
  if (fromEnv) {
    return `http://127.0.0.1:${fromEnv}`;
  }
  try {
    const j = loadMergedPortConfig();
    const port = (j.aiProvider || {}).port ?? 8010;
    return `http://127.0.0.1:${port}`;
  } catch {
    return "http://127.0.0.1:8010";
  }
}

function normalizeOrigin(raw) {
  const value = String(raw || "").trim().replace(/\/$/, "");
  if (!value) return "";
  const candidate = /^[a-z][a-z0-9+.-]*:\/\//i.test(value)
    ? value
    : `http://${value}`;
  try {
    return new URL(candidate).origin;
  } catch {
    return "";
  }
}

function readMainSaasOrigin() {
  const fromEnv = process.env.VITE_MAIN_SAAS_ORIGIN;
  if (fromEnv) return normalizeOrigin(fromEnv);
  try {
    const j = loadMergedPortConfig();
    // OPT-20260806-053: Django saas-backend 已退役（2026-07-30），主站入口迁移到 vue 前端
    const v = j.vue || {};
    const host = v.host || "127.0.0.1";
    const port = Number(v.port) || 4000;
    return `http://${host}:${port}`;
  } catch {
    return "http://127.0.0.1:4000";
  }
}

function readMainSaasSsoOrigin(kind) {
  const envName =
    kind === "admin"
      ? "VITE_MAIN_SAAS_ADMIN_SSO_ORIGIN"
      : "VITE_MAIN_SAAS_VENDOR_SSO_ORIGIN";
  const fromEnv = process.env[envName];
  if (fromEnv) return normalizeOrigin(fromEnv);
  try {
    const j = loadMergedPortConfig();
    const aiProvider = j.aiProvider || {};
    const raw =
      kind === "admin"
        ? aiProvider.adminSSODomain
        : aiProvider.vendorSSODomain;
    return normalizeOrigin(raw) || readMainSaasOrigin();
  } catch {
    return readMainSaasOrigin();
  }
}

/** 用户浏览器持有 sessionid 的主站入口（Vue dev；OPT-20260806-053: Django 已退役） */
function readMainSaasSessionOrigin() {
  const fromEnv = process.env.VITE_MAIN_SAAS_SESSION_ORIGIN;
  if (fromEnv) return normalizeOrigin(fromEnv);
  try {
    const j = loadMergedPortConfig();
    const v = j.vue || {};
    const host = v.host || "127.0.0.1";
    const port = Number(v.port) || 4000;
    return `http://${host}:${port}`;
  } catch {
    return "http://127.0.0.1:4000";
  }
}

const vitePort = Number(process.env.VITE_DEV_PORT) || 5173;
const aiProviderAllowedHosts = readAiProviderAllowedHosts();

function findMonorepoRootFromFrontend() {
  let dir = __dirname;
  for (let i = 0; i < 16; i++) {
    if (fs.existsSync(path.join(dir, "db", "registry.yaml"))) {
      return dir;
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      break;
    }
    dir = parent;
  }
  return "";
}

function saasMachineContainerSkillPath() {
  const root = findMonorepoRootFromFrontend();
  if (!root) {
    return "";
  }
  return path.join(
    root,
    "docs/skills/saas-container/saas-machine-container.md",
  );
}

function resolveSaasMachineContainerSkillDoc(versionQuery) {
  const skillPath = saasMachineContainerSkillPath();
  const v = String(versionQuery || "").trim().replace(/^[vV]/, "");
  if (!v || !skillPath) {
    return skillPath;
  }
  const dir = path.dirname(skillPath);
  const catalogPath = path.join(dir, "versions.yaml");
  if (!fs.existsSync(catalogPath)) {
    return "";
  }
  const raw = fs.readFileSync(catalogPath, "utf8");
  const docs = {};
  let pending = "";
  for (const line of raw.split("\n")) {
    const ver = line.match(/^\s+- version:\s*"?([^"\s]+)"?/);
    if (ver) {
      pending = String(ver[1]).replace(/^[vV]/, "");
      continue;
    }
    const doc = line.match(/^\s+doc:\s+(\S+)/);
    if (doc && pending) {
      docs[pending] = doc[1].replace(/['"]/g, "");
      pending = "";
    }
  }
  const name = docs[v];
  if (!name || name.includes("..") || /[\\/]/.test(name)) {
    return "";
  }
  return path.join(dir, name);
}

function serveSaasMachineContainerSkillPlugin() {
  const send = (req, res, next) => {
    const rawUrl = String(req.url || "");
    const url = rawUrl.split("?")[0];
    if (url !== "/saas-machine-container.md") {
      next();
      return;
    }
    const q = new URLSearchParams(rawUrl.includes("?") ? rawUrl.slice(rawUrl.indexOf("?") + 1) : "");
    const version = q.get("version");
    const skillPath = version
      ? resolveSaasMachineContainerSkillDoc(version)
      : saasMachineContainerSkillPath();
    if (!skillPath || !fs.existsSync(skillPath)) {
      res.statusCode = 404;
      res.end("not found");
      return;
    }
    res.setHeader("Content-Type", "text/plain; charset=utf-8");
    res.setHeader("Cache-Control", "no-cache, must-revalidate");
    fs.createReadStream(skillPath).pipe(res);
  };
  return {
    name: "serve-saas-machine-container-skill",
    configureServer(server) {
      server.middlewares.use(send);
    },
    configurePreviewServer(server) {
      server.middlewares.use(send);
    },
    writeBundle(options) {
      if (!options.dir) {
        return;
      }
      const skillPath = saasMachineContainerSkillPath();
      if (!skillPath || !fs.existsSync(skillPath)) {
        throw new Error(
          `saas-machine-container.md SSOT missing at ${skillPath || "(unresolved monorepo root)"}`,
        );
      }
      fs.copyFileSync(
        skillPath,
        path.join(options.dir, "saas-machine-container.md"),
      );
    },
  };
}

export default defineConfig({
  define: {
    "import.meta.env.VITE_MAIN_SAAS_ORIGIN": JSON.stringify(readMainSaasOrigin()),
    "import.meta.env.VITE_MAIN_SAAS_ADMIN_SSO_ORIGIN": JSON.stringify(
      readMainSaasSsoOrigin("admin"),
    ),
    "import.meta.env.VITE_MAIN_SAAS_VENDOR_SSO_ORIGIN": JSON.stringify(
      readMainSaasSsoOrigin("vendor"),
    ),
    "import.meta.env.VITE_MAIN_SAAS_SESSION_ORIGIN": JSON.stringify(
      readMainSaasSessionOrigin(),
    ),
  },
  plugins: [vue(), serveSaasMachineContainerSkillPlugin()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    ...(aiProviderAllowedHosts.length
      ? { allowedHosts: aiProviderAllowedHosts }
      : {}),
    port: vitePort,
    strictPort: true,
    host: true,
    cors: true,
    // 显式排除大目录以减少 inotify 监听消耗
    watch: {
        ignored: [
            '**/node_modules/**',
            '**/.git/**',
            '**/dist/**',
            '**/__pycache__/**',
            '**/*.pyc',
            '**/logs/**',
        ],
    },
    // 显式开启 HMR（默认即为 true；固定端口便于与 run.sh 一致）
    hmr: {
      protocol: "ws",
      host: "0.0.0.0",
      port: vitePort,
      clientPort: vitePort,
    },
    proxy: {
      "/api": { target: readAiProviderApiTarget(), changeOrigin: true },
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
