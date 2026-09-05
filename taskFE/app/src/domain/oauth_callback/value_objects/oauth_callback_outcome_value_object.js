import { OAuthCallbackCode } from './oauth_callback_code_value_object.js'
import { OAuthCallbackSeverity } from './oauth_callback_severity_value_object.js'
import { OAuthCallbackUserMessage } from './oauth_callback_user_message_value_object.js'
import { OAuthGitProvider } from './oauth_git_provider_value_object.js'

export class OAuthCallbackOutcome {
  constructor({ provider, code, severity, message }) {
    this.provider =
      provider instanceof OAuthGitProvider ? provider : new OAuthGitProvider(provider)
    this.code = code instanceof OAuthCallbackCode ? code : new OAuthCallbackCode(code)
    this.severity =
      severity instanceof OAuthCallbackSeverity
        ? severity
        : new OAuthCallbackSeverity(severity)
    this.message =
      message instanceof OAuthCallbackUserMessage
        ? message
        : new OAuthCallbackUserMessage(message)
  }

  get providerValue() {
    return this.provider.value
  }

  get codeValue() {
    return this.code.value
  }

  get severityValue() {
    return this.severity.value
  }

  get messageValue() {
    return this.message.value
  }
}
