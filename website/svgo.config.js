module.exports = {
    plugins: [
      {
        name: 'preset-default',
        params: {
          overrides: {
            removeTitle: true,
            removeViewBox: false,
            cleanupIds: {
              minify: false
            }
          },
        },
      },
    ],
  };
