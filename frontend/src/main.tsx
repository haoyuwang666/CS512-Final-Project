// TODO(Tian): Add top-level error boundaries as dashboard routes grow.
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './styles.css';
createRoot(document.getElementById('root')!).render(<StrictMode><App /></StrictMode>);
