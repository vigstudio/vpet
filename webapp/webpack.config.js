const path = require('path');

module.exports = {
  entry: './src/index.jsx',
  module: {rules: [
    {test: /\.jsx?$/, exclude: /node_modules/, use: 'babel-loader'},
    {test: /\.css$/, use: ['style-loader', 'css-loader']},
    {test: /\.(png|webp)$/, type: 'asset/inline'},
  ]},
  resolve: {extensions: ['.js', '.jsx']},
  output: {path: path.resolve(__dirname, '../cmd/vpet/static'), filename: 'main.js', clean: false},
};
