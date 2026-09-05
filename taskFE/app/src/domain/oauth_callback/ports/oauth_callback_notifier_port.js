export class OAuthCallbackNotifierPort {
  notifyError(_message) {
    throw new Error('OAuthCallbackNotifierPort.notifyError 未实现')
  }

  notifySuccess(_message) {
    throw new Error('OAuthCallbackNotifierPort.notifySuccess 未实现')
  }
}
