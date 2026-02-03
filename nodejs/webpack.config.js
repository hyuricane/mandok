const path = require('path')
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');


module.exports = {
  mode: 'production',
  entry: './src/index.js',
  output: {
    path: path.resolve(__dirname, '../static'),
    filename: 'bundle.js',
  },
  devServer: {
    static: {
      directory: path.resolve(__dirname, '../static'),
    },
    port: 8080,
    open: true,
    hot: true,
    compress: true,
    historyApiFallback: true,
  },
  module: {
    rules: [
      {
        test: /\.css$/i,
        include: path.resolve(__dirname, 'src'),
        use: [MiniCssExtractPlugin.loader, 'css-loader', 'postcss-loader'],
      },
      {
        test: /\.css$/i,
        include: /node_modules/, // Allow CSS from node_modules
        use: [
          MiniCssExtractPlugin.loader, 
          'css-loader'
        ],
      },
			{
				test: /\.ttf$/,
				type: 'asset/resource'
			}
    ],
  },
  plugins: [
    new MiniCssExtractPlugin({
      filename: "bundle.css",
    }),
    new MonacoWebpackPlugin({
      languages: ['json'],
      // customLanguages: [
      //   {
      //     label: 'yaml',
      //     entry: 'monaco-yaml',
      //     worker: {
      //       id: 'monaco-yaml/yamlWorker',
      //       entry: 'monaco-yaml/yaml.worker'
      //     }
      //   }
      // ]
    }),
    
  ]
}