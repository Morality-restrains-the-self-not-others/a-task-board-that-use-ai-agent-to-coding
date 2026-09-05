# frozen_string_literal: true
# Fix: openid_connect gem SWD.url_builder defaults to URI::HTTPS
# For HTTP issuers (taskAuth OIDC Provider), this causes SSL record layer failure.
# This initializer forces URI::HTTP so discovery uses plain HTTP.
# See: specs/oidc-sso-404-fix-design.md, gitService/scripts/fix_oidc_ssl.sh

require "swd"
SWD.url_builder = URI::HTTP
