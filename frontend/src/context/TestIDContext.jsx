import React, { createContext, useContext, useState } from 'react';

const TestIDContext = createContext();

export const TestIDProvider = ({ children }) => {
  const [testID, setTestID] = useState(null);

  return (
    <TestIDContext.Provider value={{ testID, setTestID }}>
      {children}
    </TestIDContext.Provider>
  );
};

export const useTestID = () => {
  return useContext(TestIDContext);
};
