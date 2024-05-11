import React, { useRef, useState } from 'react'
import axios from "axios"
import { useNavigate } from 'react-router-dom';
import { useTestID } from '../context/TestIDContext';
import "../styles/SubmitTest.css"

const SubmitTest = () => {
    const [inputs, setInputs] = useState({
        TestType: "",
        TestMessageDelay: 0,
        NumRequests: 0
    });
    const [err, setError] = useState(null);
    const myInputRef = useRef(null);

    const navigate = useNavigate()
    const {setTestID} = useTestID();

    const handleChange = (e) => {
        if( e.target.name === "TestType" && e.target.value === "Avalanche") {
            setInputs((prev) => ({ ...prev, TestMessageDelay: 0 }));
            const inputElement = myInputRef.current;
            if (inputElement) {
                inputElement.value = '0';
            }
        }
        setInputs((prev) => ({ ...prev, [e.target.name]: e.target.type === 'number' ? parseInt(e.target.value, 10) : e.target.value }));
    };
    const handleSubmit = async (e) => {
        e.preventDefault();
    
        try {
            const res = await axios.post("http://localhost:8080/tests", inputs, {
                headers: {
                    "Content-Type": "application/json",
                },
            });
            console.log(res);
            setTestID(res.data);
            navigate("/dashboard")
        } catch (err) {
            setError(err.message);
        }
    };    
    
    return (
        <div className = "submit-test">
            <h1>New Test</h1>
            <div className="container">
                <div className= "form">
                    <form onSubmit={handleSubmit}>
                        {/* <input required type="text" name = "TestType" placeholder = "Avalanche / Tsunami" onChange={handleChange} /> */}
                        <select required name="TestType" onChange={handleChange}>
                            <option value="" disabled selected>Select Test Type</option>
                            <option value="Avalanche">Avalanche</option>
                            <option value="Tsunami">Tsunami</option>
                        </select>
                        <input required type="number"  
                            name = "TestMessageDelay" 
                            placeholder = "Delay (ms)"
                            onChange={handleChange} 
                            disabled = {inputs.TestType === "Avalanche"}
                            ref={myInputRef}
                        />
                        <input required type="number" min="1"
                         name = "NumRequests" 
                         placeholder = "Number of Requests per Driver Node" 
                         onChange={handleChange} />
                        {err && <p className='error'> Error: {err}</p>}
                        <button type="submit">Submit Test</button>
                    </form>
                </div>
            </div>
        </div>
    );
};

export default SubmitTest;