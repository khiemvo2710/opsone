import { Configuration, PublicClientApplication } from '@azure/msal-browser';

const tenantId = import.meta.env.VITE_AAD_TENANT_ID ?? '';
const clientId = import.meta.env.VITE_AAD_WEB_CLIENT_ID ?? '';

export const devAuthBypass = import.meta.env.VITE_DEV_AUTH_BYPASS === 'true';

export const msalEnabled = !devAuthBypass && Boolean(tenantId && clientId);

export const msalConfig: Configuration = {
  auth: {
    clientId,
    authority: `https://login.microsoftonline.com/${tenantId}`,
    redirectUri: `${window.location.origin}/auth/callback`,
    postLogoutRedirectUri: `${window.location.origin}/`,
  },
  cache: { cacheLocation: 'sessionStorage', storeAuthStateInCookie: false },
};

export const loginRequest = {
  scopes: ['openid', 'profile', 'User.Read', import.meta.env.VITE_AAD_API_SCOPE ?? ''],
};

export const pca = new PublicClientApplication(msalConfig);
