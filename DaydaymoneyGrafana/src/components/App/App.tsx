import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { AppRootProps } from '@grafana/data';
const HomeRedirect = React.lazy(() => import('../../pages/HomeRedirect'));

function App(_props: AppRootProps) {
  return (
    <Routes>
      <Route path="*" element={<HomeRedirect />} />
    </Routes>
  );
}

export default App;
