import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { Navbar, Footer } from './components';
import { EditorPage, ViewPage } from './pages';

function App() {
  return (
    <BrowserRouter>
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 3000,
          style: {
            background: '#111827',
            color: '#F8FAFC',
            border: '1px solid #1E293B',
            borderRadius: '0.75rem',
          },
          success: {
            iconTheme: {
              primary: '#00D9FF',
              secondary: '#111827',
            },
          },
          error: {
            iconTheme: {
              primary: '#EF4444',
              secondary: '#111827',
            },
          },
        }}
      />
      <Navbar />
      <Routes>
        <Route path="/" element={<EditorPage />} />
        <Route path="/:id" element={<ViewPage />} />
      </Routes>
      <Footer />
    </BrowserRouter>
  );
}

export default App;
