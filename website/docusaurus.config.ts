import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const GITLAB = 'https://gitlab.com/pi-sandbox/pi-sandbox-runtime';

const config: Config = {
  title: 'PI Agent Sandbox Runtime',
  tagline: 'Local-first sandboxes for AI coding agents',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  // Vercel project Root Directory must be `website/`. Deploy is connected
  // separately; this URL is a placeholder until then.
  url: 'https://pi-sandbox-docs.vercel.app',
  baseUrl: '/',

  organizationName: 'pi-sandbox',
  projectName: 'pi-sandbox-runtime',

  onBrokenLinks: 'throw',
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/',
          editUrl: `${GITLAB}/-/tree/main/website/`,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/docusaurus-social-card.jpg',
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'PI Sandbox',
      logo: {
        alt: 'PI Sandbox Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Docs',
        },
        {
          href: GITLAB,
          label: 'GitLab',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'Introduction', to: '/'},
            {label: 'Quickstart', to: '/getting-started/quickstart'},
            {label: 'CLI reference', to: '/cli/overview'},
            {label: 'API reference', to: '/api/overview'},
          ],
        },
        {
          title: 'More',
          items: [{label: 'GitLab', href: GITLAB}],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} PI Agent Sandbox Runtime.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'go', 'json', 'yaml', 'python'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
