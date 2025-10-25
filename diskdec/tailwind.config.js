/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./*.html"], // 作用于当前目录所有HTML
  theme: {
    extend: {
      colors: {
        primary: '#165DFF', // 主色调（关键！之前的设计依赖这个颜色）
      },
    },
  },
  plugins: [],
}