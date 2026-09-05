# frozen_string_literal: true
# Unit checks for TraeGitlabTrafficQuota skip rules (no GitLab runtime).
init = ENV.fetch("TRAE_QUOTA_INIT") {
  File.expand_path("../../initializers/zzz_trae_gitlab_traffic_quota.rb", __FILE__)
}
load init

ENV["TRAE_GITLAB_PUBLIC_HOST"] = "gitlab.daydaymoney.com"
ENV["TRAE_GITLAB_INTRANET_HOSTS"] = "gitlab-internal.example.com"

def assert!(cond, msg)
  abort msg unless cond
end

mod = TraeGitlabTrafficQuota

assert!(mod.skip_quota?(from_ci: true), "CI must skip")
assert!(!mod.skip_quota?(from_ci: false), "non-CI public defaults must not skip")

# Public GitLab Host + Docker NAT (task node / port-map) must be gated.
assert!(
  !mod.skip_quota?(from_ci: false, request_host: "gitlab.daydaymoney.com", client_ip: "172.17.0.1"),
  "public Host + docker NAT must not skip"
)
assert!(
  !mod.skip_quota?(from_ci: false, request_host: "gitlab.daydaymoney.com:443", client_ip: "172.18.0.1"),
  "public Host:port + compose NAT must not skip"
)
assert!(
  !mod.skip_quota?(from_ci: false, request_host: "gitlab.daydaymoney.com", client_ip: "127.0.0.1"),
  "public Host + workhorse loopback must not skip"
)
assert!(
  !mod.skip_quota?(from_ci: false, request_host: "gitlab.daydaymoney.com", client_ip: "8.8.8.8"),
  "public Host + public client must not skip"
)

# Same-region intranet: VPC source or intranet Host / private GitLab IP.
assert!(
  mod.skip_quota?(from_ci: false, request_host: "gitlab.daydaymoney.com", client_ip: "10.88.12.40"),
  "public Host + VPC 10/8 is same-region intranet"
)
assert!(
  mod.skip_quota?(from_ci: false, request_host: "gitlab-internal.example.com", client_ip: "1.2.3.4"),
  "configured intranet Host must skip"
)
assert!(
  mod.skip_quota?(from_ci: false, request_host: "10.0.1.8", client_ip: "10.0.1.9"),
  "clone via GitLab private IP Host must skip"
)

# SSH has no Host: VPC skip, docker NAT gated.
assert!(
  mod.skip_quota?(from_ci: false, request_host: "", client_ip: "10.20.30.40"),
  "SSH from VPC must skip"
)
assert!(
  !mod.skip_quota?(from_ci: false, request_host: "", client_ip: "172.17.0.1"),
  "SSH via docker NAT must not skip"
)

StubAccess = Struct.new(:from_ci) do
  def request_from_ci_build?
    from_ci
  end
end

# GitAccessPatch path: client_ip is docker NAT (172.17/172.21), request_host is
# not resolvable outside a real request (no Gitlab::SafeRequestStore defined
# yet) → gated; CI still skips.
mod.define_singleton_method(:client_ip) { "172.17.0.1" }
assert!(!mod.skip_intranet?(StubAccess.new(false)), "access patch: docker NAT must not skip")
assert!(mod.skip_intranet?(StubAccess.new(true)), "access patch: CI must skip")

# RequestContextHostPatch path: Gitlab::RequestContext only keeps client_ip, so
# start_request_context snapshots the HTTP Host into SafeRequestStore.
# request_host must fall back to that snapshot, else intranet-Host clones
# through docker NAT would be mis-gated.
store = { trae_http_request_host: "10.2.150.68" }
unless defined?(::Gitlab)
  ::Gitlab = Module.new
end
unless defined?(::Gitlab::SafeRequestStore)
  ::Gitlab::SafeRequestStore = Class.new do
    define_singleton_method(:[]) { |k| store[k] }
  end
end
assert!(
  mod.skip_quota?(from_ci: false, request_host: mod.request_host, client_ip: "172.17.0.1"),
  "request_host snapshot: intranet Host via docker NAT must skip"
)
store[:trae_http_request_host] = "gitlab.daydaymoney.com"
assert!(
  !mod.skip_quota?(from_ci: false, request_host: mod.request_host, client_ip: "172.17.0.1"),
  "request_host snapshot: public Host via docker NAT must not skip"
)

# GitAccess 19.x: #project/#user 是 private 方法（git_access.rb:140 起 private 区段）。
# respond_to? 必须带 include_all=true，否则拿不到路径/用户名 → 闸门失效 fail-open。
PrivateAccess = Struct.new(:full_path, :username) do
  private

  def project = Struct.new(:full_path).new(full_path)

  def user = Struct.new(:username).new(username)
end
private_access = PrivateAccess.new("example-user/somanyad", "example-user")
assert!(
  mod.project_path_for(private_access) == "example-user/somanyad",
  "project_path_for must read private GitAccess#project (respond_to?(sym, true))"
)
assert!(
  mod.gitlab_username_for(private_access) == "example-user",
  "gitlab_username_for must read private GitAccess#user (respond_to?(sym, true))"
)
# 匿名/缺方法对象保持空串（fail-open 语义）
assert!(mod.gitlab_username_for(Object.new) == "", "anonymous access must yield empty username")
assert!(mod.project_path_for(Object.new) == "", "opaque access must yield empty path")

puts "PASS skip_quota: CI + VPC/intranet-Host skip; public Host + docker NAT gated; request_host snapshot"
