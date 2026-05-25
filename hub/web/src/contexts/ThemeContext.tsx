import React, { createContext, useContext, useEffect, useState } from 'react';

type Theme = 'light' | 'dark' | 'auto';

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  currentDateTheme: 'light' | 'dark'; // The actual active theme after resolving 'auto'
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export const ThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [theme, setThemeState] = useState<Theme>(() => {
    const saved = localStorage.getItem('rclone-backup-theme');
    return (saved as Theme) || 'auto';
  });

  const [currentDateTheme, setCurrentDateTheme] = useState<'light' | 'dark'>('dark');

  useEffect(() => {
    localStorage.setItem('rclone-backup-theme', theme);
  }, [theme]);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

    const updateTheme = () => {
      if (theme === 'auto') {
        const newTheme = mediaQuery.matches ? 'dark' : 'light';
        setCurrentDateTheme(newTheme);
        document.documentElement.setAttribute('data-theme', newTheme);
      } else {
        setCurrentDateTheme(theme);
        document.documentElement.setAttribute('data-theme', theme);
      }
    };

    updateTheme();

    mediaQuery.addEventListener('change', updateTheme);
    return () => mediaQuery.removeEventListener('change', updateTheme);
  }, [theme]);

  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme);
  };

  return (
    <ThemeContext.Provider value={{ theme, setTheme, currentDateTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};