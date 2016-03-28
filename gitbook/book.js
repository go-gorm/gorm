var pkg = require('./package.json');

module.exports = {
    // Documentation for GitBook is stored under "docs"
    root: './docs',
    title: 'GitBook Toolchain Documentation',

    // Enforce use of GitBook v3
    gitbook: pkg.version,

    // Use the "official" theme
    plugins: ['theme-official', 'sitemap'],
    theme: 'official',

    variables: {
        version: pkg.version
    },

    pluginsConfig: {
        sitemap: {
            hostname: 'https://docs.gitbook.com'
        }
    }
};
