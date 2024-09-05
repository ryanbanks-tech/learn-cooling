import React, { useEffect } from 'react';
import './App.css';

function App() {
    useEffect(() => {
        createSnowflakes();
    }, []);

    const createSnowflakes = () => {
        const snowContainer = document.getElementById('snow-container');
        const snowflakeCount = 50; // Adjust the number of snowflakes

        for (let i = 0; i < snowflakeCount; i++) {
            const snowflake = document.createElement('div');
            snowflake.className = 'snowflake';
            snowflake.style.left = `${Math.random() * 100}%`;
            snowflake.style.animationDuration = `${Math.random() * 3 + 2}s`; // Random duration between 2s and 5s
            snowflake.style.animationDelay = `${Math.random() * 5}s`; // Random delay
            snowContainer.appendChild(snowflake);
        }
    };

    return (
        <div id="snow-container">
            <div className="flash-text-container">
                <h1 className="flash-text">COOL LEARNING COMING SOON!!</h1>
            </div>
        </div>
    );
}

export default App;
