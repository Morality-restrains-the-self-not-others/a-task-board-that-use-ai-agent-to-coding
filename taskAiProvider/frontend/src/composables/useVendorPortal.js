/**
 * VendorPortal facade — wires split composables (OPT-20260817-016).
 */
import { onMounted, reactive } from "vue";
import { initVendorPortalSession } from "./useVendorPortalSession.js";
import { initVendorPortalImageGroups } from "./useVendorPortalImageGroups.js";
import { initVendorPortalCloudServers } from "./useVendorPortalCloudServers.js";

export function useVendorPortal() {
  const d = {};
  initVendorPortalSession(d);
  initVendorPortalImageGroups(d);
  initVendorPortalCloudServers(d);

  onMounted(async () => {
    d.clearMsg();
    if (d.tryExchangeOidcFromHash()) {
      try {
        await d.loadMe();
        await d.refreshGroups();
        await d.refreshImages();
        await d.refreshServers();
        await d.loadCredentials();
        await d.loadUserdataTemplateOptions();
        return;
      } catch (e) {
        d.clearLocalVendorSession();
        d.setMsgError(e);
        return;
      }
    }
    try {
      if (await d.tryExchangeSsoFromHash()) {
        await d.loadMe();
        await d.refreshGroups();
        await d.refreshImages();
        await d.refreshServers();
        await d.loadCredentials();
        await d.loadUserdataTemplateOptions();
        return;
      }
    } catch (e) {
      d.setMsgError(e);
    }
    if (d.token.value) {
      try {
        await d.loadMe();
        await d.refreshGroups();
        await d.refreshImages();
        await d.refreshServers();
        await d.loadCredentials();
        await d.loadUserdataTemplateOptions();
      } catch {
        d.clearLocalVendorSession();
      }
    }
  });

  return reactive(d);
}
