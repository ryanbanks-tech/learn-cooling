import React from 'react';
import { createRoot } from 'react-dom/client';
import { InertiaApp } from '@inertiajs/inertia-react';

const App = () => (
    <InertiaApp
        initialPage={JSON.parse(document.getElementById('app').dataset.page)}
        resolveComponent={(name) => import(`./pages/${name}`).then(module => module.default)}
    />
);

const root = createRoot(document.getElementById('app'));
root.render(<App />);
