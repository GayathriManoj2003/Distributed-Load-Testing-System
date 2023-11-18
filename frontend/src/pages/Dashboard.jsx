import React, { useState, useEffect } from 'react';
// import { useTestID } from '../context/TestIDContext';

const Dashboard = () => {
  const [messages, setMessages] = useState([]);

  // const {testID} = useTestID()

  useEffect(() => {
    const socket = new WebSocket('ws://localhost:8080/ws');

    socket.onmessage = (event) => {
      const newMessages = [...messages, event.data];
      setMessages(newMessages);
    };

    return () => {
      // Clean up the WebSocket connection when the component
      // unmounts
      socket.close();
    };
  }, [messages]);

  return (
    <div>
      <h1>WebSocket Messages</h1>
      <ul>
        {messages.map((message, index) => (
          <li key={index}>{message}</li>
        ))}
      </ul>
    </div>
  );
};

export default Dashboard;
