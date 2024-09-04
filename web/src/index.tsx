// import React from 'react';
// import ReactDOM from 'react-dom/client';
// import './index.css';
// import App from './App';
// import reportWebVitals from './reportWebVitals';

// const root = ReactDOM.createRoot(
//   document.getElementById('root') as HTMLElement
// );
// root.render(
//   <React.StrictMode>
//     <App />
//   </React.StrictMode>
// );

// // If you want to start measuring performance in your app, pass a function
// // to log results (for example: reportWebVitals(console.log))
// // or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
// reportWebVitals();

import React from 'react';
import Welcome from './pages/Welcome';
import { createRoot } from 'react-dom/client';
import { InertiaApp } from '@inertiajs/inertia-react';

const App = () => (
  <InertiaApp
    initialPage={JSON.parse(document.getElementById('app')?.dataset.page!)}
    resolveComponent={(name) => import(`./pages/${name}`).then(module => module.default)}
    initialComponent={<Welcome />} // Wrap the Welcome component inside JSX element
  />
);

const root = createRoot(document.getElementById('app')!);
root.render(<App />);
