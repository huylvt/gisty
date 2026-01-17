import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { Navbar } from './components';
import { EditorPage, ViewPage } from './pages';

function App() {
  return (
    <BrowserRouter>
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 3000,
          style: {
            background: '#1E293B',
            color: '#F8FAFC',
            border: '1px solid #334155',
          },
          success: {
            iconTheme: {
              primary: '#00D1FF',
              secondary: '#1E293B',
            },
          },
          error: {
            iconTheme: {
              primary: '#EF4444',
              secondary: '#1E293B',
            },
          },
        }}
      />
      <Navbar />
      <Routes>
        <Route path="/" element={<EditorPage />} />
        <Route path="/:id" element={<ViewPage />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
