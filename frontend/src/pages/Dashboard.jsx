import axios from 'axios';
import React, { useState, useEffect } from 'react';
import { useTestID } from '../context/TestIDContext';
import "../styles/Dashboard.scss"
import { useNavigate } from 'react-router-dom';

const Dashboard = () => {
  const [nodeStats, setNodeStats] = useState({});
  const [finalStats, setFinalStats] = useState({});
  const { testID, setTestID} = useTestID();
  const [curTestID, setCurTestID] = useState();

  const navigate = useNavigate();
  useEffect(() => {
    if(testID != null) {
      setCurTestID(testID)
      setFinalStats(null)
      trigger();
    } else {
      navigate("/");
    }
    // eslint-disable-next-line
  }, []);

  const trigger = async () => {
    if(testID != null) {
      try {
        await axios.post("http://localhost:8080/trigger", {"TestID": testID});
        // const res = await axios.post("http://localhost:8080/trigger");
        // console.log(testID);
        // console.log(res);
      } catch (err) {
        console.log(err);
      }
    }
  };

  useEffect(() => {
    const socket = new WebSocket('ws://localhost:8080/ws');

    socket.onmessage = (event) => {
      const parsed_json = JSON.parse(event.data);
      const numKeyValuePairs = Object.keys(parsed_json).length;


      if(numKeyValuePairs === 2) {
        console.log(numKeyValuePairs, parsed_json)
        const {test_id, metrics} = parsed_json;
        if(test_id === testID) {
          setFinalStats(metrics)
          console.log("Final", metrics)
          setTestID(null)
          console.log(finalStats)
          return () => {
            socket.close();
          };
        }
      }
      if(numKeyValuePairs === 1) {
        console.log(numKeyValuePairs, parsed_json)
        const {NodeID} = parsed_json;
        setNodeStats((prevNodeStats) => {
          // Update only the status attribute for the specific NodeID
          const updatedNodeStats = {
            ...prevNodeStats,
            [NodeID]: {
              ...prevNodeStats[NodeID],
              status: 0,
            },
          };
          return updatedNodeStats;
        });
      } else {
        const { TestID, NodeID, TotalRequests, MeanLatency, MinLatency, MaxLatency } = parsed_json;
        // console.log(parsed_json)
        if (testID === TestID) {
          setNodeStats((prevNodeStats) => {
            // Update the stats for the specific NodeID
            return {
              ...prevNodeStats,
              [NodeID]: {
                mean: parseFloat(MeanLatency).toFixed(2),
                min: MinLatency,
                max: MaxLatency,
                total: TotalRequests,
                status: 1,
              },
            };
          });
        }
      }
    };

    return () => {
      // Clean up the WebSocket connection when the component unmounts
      socket.close();
    };
    // eslint-disable-next-line
  }, [testID]);

  return (
    <div className='dashboard'>
      <h1>Test Results</h1>
      <h2> Test ID: {curTestID} </h2>
      {finalStats ? <h3> Test Complete </h3> : <h3> Test in Progress</h3>}
      <div>
        {finalStats && (
          <div className='finalstats'>
            <h2> Final Statistics </h2>
            <p>Minimum Latency: {finalStats.MinLatency}</p>
            <p>Maximum Latency: {finalStats.MaxLatency}</p>
            <p>Mean Latency: {finalStats.MeanLatency}</p> {/* Corrected property name */}
            <p>Number of Driver Nodes: {finalStats.NumNodes}</p>
          </div>
        )}
      </div>
      {finalStats && (<button className= "nav-button"
      onClick={() => navigate("/")}>Go Back To Home</button>)}  
      <div className='nodestats'>
        <h2>Driver Node Statistics</h2>
        {Object.keys(nodeStats).map((nodeID) => (
          <div
          key={nodeID}
          className={`nodeDetails ${nodeStats[nodeID].status === 0 ? 'greyed-out' : ''}`}
        >
            <span>Node ID: {nodeID}   {!nodeStats[nodeID].status && (<span>---Inactive</span>)}</span>
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
