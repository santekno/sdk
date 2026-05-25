import type { DocsThemeConfig } from 'nextra-theme-docs'

const config: DocsThemeConfig = {
  logo: <strong>Santekno Go SDK</strong>,
  project: {
    link: 'https://github.com/santekno/sdk',
  },
  docsRepositoryBase: 'https://github.com/santekno/sdk/tree/main/docs',
  footer: {
    text: '© 2026 Santekno Maintainers — MIT License',
  },
  primaryHue: 172,  // #00C2A8 hue
  useNextSeoProps() {
    return {
      titleTemplate: '%s – Santekno Go SDK',
    }
  },
}

export default config
