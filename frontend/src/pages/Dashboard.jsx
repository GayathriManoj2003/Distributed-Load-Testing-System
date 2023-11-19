import axios from 'axios';
import React, { useState, useEffect } from 'react';
import { useTestID } from '../context/TestIDContext';
import "./Dashboard.scss"

const Dashboard = () => {
  const [nodeStats, setNodeStats] = useState({});
  const { testID, setTestID} = useTestID();

  const CurTestID = testID
  useEffect(() => {
    trigger();
    // eslint-disable-next-line
  }, []);

  const trigger = async () => {
    try {
      await axios.post("http://localhost:8080/trigger", {"TestID": testID});
      // const res = await axios.post("http://localhost:8080/trigger");
      // console.log(testID);
      // console.log(res);
    } catch (err) {
      console.log(err);
    }
  };

  useEffect(() => {
    const socket = new WebSocket('ws://localhost:8080/ws');

    socket.onmessage = (event) => {
      const parsed_json = JSON.parse(event.data);
      const { TestID, NodeID, TotalRequests, MeanLatency, MinLatency, MaxLatency } = parsed_json;
      console.log(parsed_json)
      if (testID === TestID) {
        setNodeStats((prevNodeStats) => {
          // Update the stats for the specific NodeID
          return {
            ...prevNodeStats,
            [NodeID]: {
              mean: MeanLatency/TotalRequests,
              min: MinLatency,
              max: MaxLatency,
              total: TotalRequests,
            },
          };
        });
      }
    };

    return () => {
      // Clean up the WebSocket connection when the component unmounts
      socket.close();
    };
  }, [testID]);

  return (
    <div className='dashboard'>
      <h1>Test Results</h1>
      <h2> Test ID: {testID} </h2>
      <div className='nodestats'>
        <h2>Driver Node Statistics</h2>
        {Object.keys(nodeStats).map((nodeID) => (
          <div key={nodeID} className='nodeDetails'>
            <span>Node ID: {nodeID}</span>
            <div className='metrics'>
              <p>Mean Latency: {nodeStats[nodeID].mean}</p>
              <p>Min Latency: {nodeStats[nodeID].min}</p>
              <p>Max Latency: {nodeStats[nodeID].max}</p>
              <p>Total Requests: {nodeStats[nodeID].total}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default Dashboard;
