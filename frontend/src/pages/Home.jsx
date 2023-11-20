import React from 'react';
import { Link } from 'react-router-dom';
import './Home.css';

const Home = () => {
  return (
    <div className="home-container">
      <h1>Distributed Load Testing</h1>
      <p>Welcome to our Distributed Load Testing System website! 
        As part of our Big Data Project at PES University, we've developed
         a cutting-edge system designed for distributed load testing of http servers. 
         Our architecture, featuring Orchestrator and Driver nodes, 
         leverages Kafka for seamless communication.
          Users can choose between Tsunami and Avalanche testing modes, 
          tailoring their load tests.
           With dynamic node registration and the ability to specify number of requests per driver node,
            our system offers flexibility and scalability.
         Experience the future of web server load testing—efficient,
          dynamic, and orchestrated seamlessly through Kafka. 
          Explore our site to get started!</p>
      <Link to="/submit_test">
        <button className="new-test-button">New Test</button>
      </Link>
    </div>
  );
};

export default Home;
