# v116 enterprise-landscape — 邮件邀请退订 (target)

```mermaid
graph TD;
  invitee["被邀请邮箱持有人"];
  inviter["邀请人"];
  clickUnsub["点击邮件退订"];
  copyLink["复制邀请链接给对方"];
  fe["taskFE"];
  auth["taskAuth"];
  p115["Plateau v115"];
  p116["Plateau v116"];
  g["Gap: 无法退订邀请邮件"];
  wp["WP-email-invite-unsubscribe"];
  invitee --> clickUnsub;
  inviter --> copyLink;
  invitee --> fe;
  fe --> auth;
  inviter --> fe;
  p115 --> g;
  wp --|> g;
  wp --|> p116;
```
