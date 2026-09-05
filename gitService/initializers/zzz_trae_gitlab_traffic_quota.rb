# frozen_string_literal: true
# Trae: deny git-upload-pack when tenant prepaid GitLab traffic is exhausted.
# ADR-0036. Fail-open if taskBill is unreachable. Direct HTTP (no env proxy).
#
# Intranet skip is NOT RFC1918 client IP (Docker NAT / Workhorse loopback
# would false-skip public clones from task nodes). Skip only: CI, configured
# intranet Host, GitLab private-IP Host, or true VPC source (10/8, 192.168/16).

require "json"
require "net/http"
require "uri"
require "ipaddr"

module TraeGitlabTrafficQuota
  CACHE_TTL = 15
  TIMEOUT_SEC = 2
  DEFAULT_MSG = "GitLab traffic quota exceeded. Purchase prepaid traffic on the tenant GitLab settings page."
  DOCKER_OR_LOOPBACK = [
    IPAddr.new("127.0.0.0/8"),
    IPAddr.new("::1"),
    IPAddr.new("169.254.0.0/16"),
    IPAddr.new("172.16.0.0/12")
  ].freeze
  VPC_INTRANET = [
    IPAddr.new("10.0.0.0/8"),
    IPAddr.new("192.168.0.0/16")
  ].freeze

  module GitAccessPatch
    def check_download_access!
      super
      TraeGitlabTrafficQuota.enforce!(self)
    end
  end

  # Gitlab::RequestContext only stores client_ip (see gitlab/request_context.rb:
  # start_request_context builds a Rack::Request but drops it). Snapshot the
  # HTTP Host there so the gate can tell public clones from same-region
  # intranet Host clones (Docker NAT makes client_ip 172.17/172.21 either way).
  module RequestContextHostPatch
    def start_request_context(request:)
      super
      Gitlab::SafeRequestStore[:trae_http_request_host] = begin
        host = Rack::Request.new(request.env).host.to_s.strip.downcase
        host.empty? ? nil : host
      rescue StandardError
        nil
      end
    end
  end

  class << self
    def enforce!(access)
      return unless download_command?(access)
      from_ci = access.respond_to?(:request_from_ci_build?) && access.request_from_ci_build?
      host = request_host_for(access)
      ip = client_ip
      return if skip_quota?(from_ci: from_ci, request_host: host, client_ip: ip)

      if private_ip?(ip)
        warn_log("trae_gitlab_traffic_gate_private_ip_enforced client_ip=#{ip} host=#{host} from_ci=#{from_ci}")
      end

      result = check(
        project_path: project_path_for(access),
        from_ci: from_ci,
        is_intranet: intranet_request?(request_host: host, client_ip: ip),
        gitlab_username: gitlab_username_for(access)
      )
      return unless result.is_a?(Hash) && result["allowed"] == false

      raise ::Gitlab::GitAccess::ForbiddenError, (result["message"].presence || DEFAULT_MSG)
    rescue ::Gitlab::GitAccess::ForbiddenError
      raise
    rescue StandardError => e
      warn_log("trae_gitlab_traffic_gate_failed error=#{e.class}: #{e.message}")
      nil
    end

    def download_command?(access)
      cmd = access.respond_to?(:cmd) ? access.cmd.to_s : ""
      return true if cmd.empty?

      ::Gitlab::GitAccess::DOWNLOAD_COMMANDS.include?(cmd)
    rescue StandardError
      true
    end

    def skip_quota?(from_ci:, request_host: nil, client_ip: nil)
      return true if from_ci == true

      intranet_request?(request_host: request_host, client_ip: client_ip)
    end

    def skip_intranet?(access)
      from_ci = access.respond_to?(:request_from_ci_build?) && access.request_from_ci_build?
      skip_quota?(
        from_ci: from_ci,
        request_host: request_host_for(access),
        client_ip: client_ip
      )
    end

    def intranet_request?(request_host:, client_ip:)
      host = normalize_host(request_host)
      pub = public_gitlab_host
      if !host.empty? && intranet_configured_hosts.include?(host) && host != pub
        return true
      end
      return true if !host.empty? && vpc_intranet_ip?(host)

      ip = client_ip.to_s
      return false if ip.empty? || false_intranet_ip?(ip)

      vpc_intranet_ip?(ip)
    end

    def request_host_for(access)
      proto = access.respond_to?(:protocol) ? access.protocol.to_s : ""
      return "" if proto == "ssh"

      request_host
    end

    def request_host
      if defined?(::Gitlab::SafeRequestStore) && ::Gitlab::SafeRequestStore.respond_to?(:[])
        host = ::Gitlab::SafeRequestStore[:trae_http_request_host]
        return normalize_host(host) unless host.nil? || host.empty?
      end
      if defined?(RequestStore) && RequestStore.respond_to?(:store)
        req = RequestStore.store[:request]
        if req.respond_to?(:host)
          host = normalize_host(req.host)
          return host unless host.empty?
        end
      end
      candidates = []
      if defined?(::Gitlab::RequestContext) && ::Gitlab::RequestContext.respond_to?(:instance)
        ctx = ::Gitlab::RequestContext.instance
        if ctx.respond_to?(:request) && ctx.request
          req = ctx.request
          candidates << req.host if req.respond_to?(:host)
          if req.respond_to?(:get_header)
            candidates << req.get_header("HTTP_HOST")
          elsif req.respond_to?(:env) && req.env
            candidates << req.env["HTTP_HOST"]
          end
        end
      end
      candidates.each do |raw|
        host = normalize_host(raw)
        return host unless host.empty?
      end
      ""
    rescue StandardError
      ""
    end

    def client_ip
      return unless defined?(::Gitlab::RequestContext) && ::Gitlab::RequestContext.respond_to?(:instance)

      ::Gitlab::RequestContext.instance.client_ip
    rescue StandardError
      nil
    end

    def private_ip?(raw)
      return false if raw.nil? || raw.to_s.empty?

      addr = IPAddr.new(raw.to_s)
      addr.loopback? || addr.private? || (addr.respond_to?(:link_local?) && addr.link_local?)
    rescue IPAddr::Error
      false
    end

    def false_intranet_ip?(raw)
      addr = IPAddr.new(raw.to_s)
      return true if addr.loopback?
      return true if addr.respond_to?(:link_local?) && addr.link_local?

      DOCKER_OR_LOOPBACK.any? { |cidr| cidr.include?(addr) }
    rescue IPAddr::Error
      %w[localhost].include?(raw.to_s.downcase)
    end

    def vpc_intranet_ip?(raw)
      addr = IPAddr.new(raw.to_s)
      return false if false_intranet_ip?(raw)

      VPC_INTRANET.any? { |cidr| cidr.include?(addr) }
    rescue IPAddr::Error
      false
    end

    def normalize_host(raw)
      s = raw.to_s.strip.downcase
      return "" if s.empty?

      s = s.sub(/\A\[([^\]]+)\]/, '\1')
      s = s.split(":", 2).first if s.count(":") == 1
      s
    end

    def public_gitlab_host
      env = ENV["TRAE_GITLAB_PUBLIC_HOST"].to_s
      host = normalize_host(env)
      return host unless host.empty?
      return normalize_host(Gitlab.config.gitlab.host) if defined?(Gitlab) && Gitlab.respond_to?(:config)

      ""
    rescue StandardError
      ""
    end

    def intranet_configured_hosts
      ENV["TRAE_GITLAB_INTRANET_HOSTS"].to_s.split(/[,\s]+/).map { |h| normalize_host(h) }.reject(&:empty?)
    end

    # 认证用户的 GitLab 用户名：个人命名空间仓（user/repo）由 taskBill 按仓所有者
    # 归租户计费；组仓（group/repo）无 tenant-{id} 前缀时按认证用户归租户（OPT-20260824-079）。
    # 匿名请求返回空串，taskBill 回退到 project_path 顶层命名空间。
    # GitLab 19.x 中 GitAccess#user/#project 是 private 方法，respond_to? 必须带
    # include_all=true（否则返回 false → 拿不到路径/用户名 → 闸门失效 fail-open，
    # 线上症状：个人命名空间仓与租户组仓在 19.x 上一律 UNMAPPED）。
    def gitlab_username_for(access)
      user = access.respond_to?(:user, true) ? access.send(:user) : nil
      user.respond_to?(:username) ? user.username.to_s.strip : ""
    rescue StandardError
      ""
    end

    def project_path_for(access)
      project = access.respond_to?(:project, true) ? access.send(:project) : nil
      return project.full_path.to_s if project.respond_to?(:full_path)

      path = access.respond_to?(:repository_path) ? access.repository_path.to_s : ""
      path.sub(/\.git\z/, "")
    rescue StandardError
      ""
    end

    def check(project_path:, from_ci:, is_intranet: false, gitlab_username: "")
      base = ENV["TRAE_TASKBILL_BASE"].to_s.strip
      return { "allowed" => true, "code" => "GATE_UNCONFIGURED" } if base.empty?

      # 缓存键含用户名：个人命名空间仓按认证用户归租户，不同用户可能落不同租户配额。
      key = "trae-traffic-gate:#{ENV.fetch('TRAE_GITLAB_REGION', '')}:#{project_path}:#{gitlab_username}:#{from_ci}:#{is_intranet}"
      if defined?(Rails) && Rails.respond_to?(:cache)
        return Rails.cache.fetch(key, expires_in: CACHE_TTL) { http_check(base, project_path, from_ci, is_intranet, gitlab_username) }
      end

      http_check(base, project_path, from_ci, is_intranet, gitlab_username)
    end

    def http_check(base, project_path, from_ci, is_intranet, gitlab_username)
      uri = URI.parse("#{base.to_s.chomp('/')}/api/internal/taskbill/gitlab-traffic-gate/")
      req = Net::HTTP::Post.new(uri)
      req["Content-Type"] = "application/json"
      req["Accept"] = "application/json"
      secret = ENV["TRAE_TASKBILL_INTERNAL_SECRET"].to_s
      req["X-TaskBill-Internal-Secret"] = secret unless secret.empty?
      req.body = {
        project_path: project_path,
        region: ENV["TRAE_GITLAB_REGION"].to_s,
        from_ci: from_ci,
        is_intranet: is_intranet,
        gitlab_username: gitlab_username.to_s
      }.to_json

      http = Net::HTTP.new(uri.host, uri.port, nil, nil)
      http.use_ssl = (uri.scheme == "https")
      http.open_timeout = TIMEOUT_SEC
      http.read_timeout = TIMEOUT_SEC
      res = http.request(req)
      JSON.parse(res.body)
    rescue StandardError => e
      warn_log("trae_gitlab_traffic_gate_unreachable error=#{e.class}: #{e.message}")
      { "allowed" => true, "code" => "GATE_UNREACHABLE" }
    end

    def warn_log(msg)
      if defined?(Rails) && Rails.respond_to?(:logger) && Rails.logger
        Rails.logger.warn(msg)
      else
        warn(msg)
      end
    end
  end
end

if defined?(Gitlab::GitAccess)
  Gitlab::GitAccess.prepend(TraeGitlabTrafficQuota::GitAccessPatch)
end
if defined?(Gitlab::RequestContext) && defined?(Gitlab::SafeRequestStore)
  Gitlab::RequestContext.singleton_class.prepend(TraeGitlabTrafficQuota::RequestContextHostPatch)
end
