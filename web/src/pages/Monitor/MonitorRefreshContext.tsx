import React, { createContext, useContext, useState, useEffect } from 'react';

interface MonitorRefreshContextType {
  onRefresh: (() => void) | null;
  setOnRefresh: (fn: (() => void) | null) => void;
  loading: boolean;
  setLoading: (loading: boolean) => void;
}

const MonitorRefreshContext = createContext<MonitorRefreshContextType | undefined>(undefined);

export const MonitorRefreshProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [onRefresh, setOnRefresh] = React.useState<(() => void) | null>(null);
  const [loading, setLoading] = React.useState(false);

  const value = React.useMemo(() => ({
    onRefresh,
    setOnRefresh,
    loading,
    setLoading
  }), [onRefresh, loading]);

  return (
    <MonitorRefreshContext.Provider value={value}>
      {children}
    </MonitorRefreshContext.Provider>
  );
};

export const useMonitorRefresh = () => {
  const context = useContext(MonitorRefreshContext);
  if (context === undefined) {
    throw new Error('useMonitorRefresh must be used within a MonitorRefreshProvider');
  }
  return context;
};

export const useRegisterMonitorRefresh = (refreshFn: () => void, loading: boolean) => {
  const { setOnRefresh, setLoading } = useMonitorRefresh();

  useEffect(() => {
    setOnRefresh(() => refreshFn);
    return () => setOnRefresh(null);
  }, [refreshFn, setOnRefresh]);

  useEffect(() => {
    setLoading(loading);
  }, [loading, setLoading]);
};
