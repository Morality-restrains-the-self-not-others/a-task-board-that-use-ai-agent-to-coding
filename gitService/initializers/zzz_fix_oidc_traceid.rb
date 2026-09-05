# frozen_string_literal: true
# Fix: Inject data-traceId into OIDC OmniAuth error flash alerts.
#
# When taskAuth redirects back with an OIDC error, it includes trace_id as
# a query parameter so the trace can be correlated with Loki/Grafana logs.
# This initializer:
#   1. Extracts trace_id from callback params on OmniAuth failure
#   2. Appends [traceId: xxx] to the visible error text
#   3. Injects a small inline <script> via flash[:raw] that adds
#      data-traceId attribute to the gl-alert div on the sign-in page.
#
# See: memory/oidc-*.md, taskAuth/src/oidc_handlers.go sendError closure

module OmniAuthTraceId
  TRACE_ID_PARAM = 'trace_id'.freeze

  def self.extract(params, request_env)
    # 1. Direct query param (taskAuth sends this in error redirects)
    tid = params[TRACE_ID_PARAM]
    return tid if tid.present?

    # 2. Nested under omniauth.params
    omniauth_params = request_env&.dig('omniauth.params') || {}
    tid = omniauth_params[TRACE_ID_PARAM]
    return tid if tid.present?

    # 3. From error response JSON body (taskAuth v2)
    error = request_env&.dig('omniauth.error')
    if error.respond_to?(:response) && error.response.respond_to?(:body)
      begin
        require 'json' unless defined?(JSON)
        data = JSON.parse(error.response.body.to_s)
        tid = data[TRACE_ID_PARAM] || data['trace_id']
      rescue StandardError => e
        # Log raw error response body for debugging when trace_id extraction fails
        # (e.g. gateway returns HTML error page instead of JSON).
        body_snippet = error.response.body.to_s.truncate(1024) rescue '(unavailable)'
        Rails.logger.warn "[OmniAuthTraceId] Failed to parse error response body: #{e.class} — status=#{error.response.status rescue '?'} body=#{body_snippet}"
        nil
      end
      return tid if tid.present?
    end

    nil
  end

  def self.inject_script(tid)
    <<~JS.html_safe
      <script>
      (function(){
        var tid=#{tid.to_json};
        if(!tid)return;
        var el=document.querySelector('.flash-container .gl-alert[data-testid="alert-danger"]');
        if(el){el.setAttribute('data-traceId',tid);}
        el=document.querySelector('.gl-alert.flash-alert');
        if(el){el.setAttribute('data-traceId',tid);}
      })();
      </script>
    JS
  end
end

Rails.application.config.after_initialize do
  next unless defined?(OmniauthCallbacksController)

  # Patch the failure method to capture trace_id from OIDC error redirects.
  OmniauthCallbacksController.class_eval do
    after_action :inject_omniauth_trace_id, only: [:failure]

    private

    def inject_omniauth_trace_id
      tid = OmniAuthTraceId.extract(params, request.env)
      return if tid.blank?

      # Append traceId to the visible error message so users can report it.
      if flash[:alert].present? && !flash[:alert].to_s.include?(tid)
        flash[:alert] = "#{flash[:alert]} [traceId: #{tid}]"
      end

      # Inject a script via flash[:raw] that adds data-traceId attribute
      # to the gl-alert div on the sign-in page.
      script = OmniAuthTraceId.inject_script(tid)
      flash[:raw] = [flash[:raw], script].compact.join("\n").html_safe
    end
  end
end
