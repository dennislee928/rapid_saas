import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ember: {
          50: "#fff5ed",
          400: "#ff8a3d",
          500: "#ff681f",
          700: "#b63816"
        },
        ink: {
          950: "#07100f",
          900: "#0b1716",
          800: "#12211f"
        },
        lichen: {
          100: "#e8f4dc",
          300: "#a8cf84",
          500: "#6e9f45"
        }
      },
      fontFamily: {
        display: ["Aptos Display", "Iowan Old Style", "Georgia", "serif"],
        body: ["Aptos", "Gill Sans", "Trebuchet MS", "sans-serif"],
        mono: ["Berkeley Mono", "SFMono-Regular", "Consolas", "monospace"]
      },
      boxShadow: {
        glow: "0 24px 80px rgba(255, 104, 31, 0.22)",
        panel: "0 24px 80px rgba(3, 8, 7, 0.38)"
      }
    }
  },
  plugins: []
};

export default config;
