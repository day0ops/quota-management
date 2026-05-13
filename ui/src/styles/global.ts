import { css } from '@emotion/react';
import { colors, darkTheme, lightTheme } from './colors';
import { fontFamily } from './typography';

// Generate CSS variable declarations from theme object
const themeToCSS = (theme: Record<string, string>) =>
  Object.entries(theme)
    .map(([key, value]) => `${key}: ${value};`)
    .join('\n    ');

export const globalStyles = css`
  :root,
  [data-theme='dark'] {
    ${themeToCSS(darkTheme)}
  }

  [data-theme='light'] {
    ${themeToCSS(lightTheme)}
  }

  *,
  *::before,
  *::after {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  html {
    font-size: 16px;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }

  body {
    font-family: ${fontFamily};
    background-color: ${colors.background};
    color: ${colors.foreground};
    line-height: 1.5;
    min-height: 100vh;
    transition:
      background-color 0.2s ease,
      color 0.2s ease;
  }

  #root {
    min-height: 100vh;
    display: flex;
  }

  a {
    color: inherit;
    text-decoration: none;
  }

  button {
    font-family: inherit;
    cursor: pointer;
    border: none;
    background: none;
  }

  input,
  textarea,
  select {
    font-family: inherit;
  }

  table {
    border-collapse: collapse;
  }

  ::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }

  ::-webkit-scrollbar-track {
    background: ${colors.background};
  }

  ::-webkit-scrollbar-thumb {
    background: ${colors.border};
    border-radius: 4px;
  }

  ::-webkit-scrollbar-thumb:hover {
    background: ${colors.borderLight};
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes fadeInScale {
    from {
      opacity: 0;
      transform: scale(0.97);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  @keyframes fg-draw {
    to {
      stroke-dashoffset: 0;
    }
  }

  @keyframes fg-fade-up {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes fg-pop-in {
    from {
      opacity: 0;
      transform: scale(0);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  @keyframes fg-pulse {
    0%,
    100% {
      opacity: 0.9;
      transform: scale(1);
    }
    50% {
      opacity: 0.2;
      transform: scale(2.2);
    }
  }
`;
