import React, { useState } from 'react'
import axios from "axios"
import { useNavigate } from 'react-router-dom';

const CreateTest = () => {
    const [inputs, setInputs] = useState({
        NumberNodes: 0,
        TestType: "",
        TestMessageDelay: 0
    });
    const [err, setError] = useState(null);

    const navigate = useNavigate()

    const handleChange = (e) => {
        setInputs((prev) => ({ ...prev, [e.target.name]: e.target.type === 'number' ? parseInt(e.target.value, 10) : e.target.value }));
    };
    const handleSubmit = async (e) => {
        e.preventDefault();
        console.log(inputs);
    
        try {
            const res = await axios.post("http://localhost:8080/test_config", inputs, {
                headers: {
                    "Content-Type": "application/json",
                },
            });
            console.log(res);
            navigate("/start_test")
        } catch (err) {
            setError(err.message);
        }
    };    
    
    return (
        <div>
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
                        <input required type="number"  name = "TestMessageDelay" placeholder = "Delay (ms)" onChange={handleChange} />
                        <input required type="number" min="2" max="8" name = "NumberNodes" placeholder = "Number of Driver Nodes (2 - 8) " onChange={handleChange} />
                        {err && <p className='error'> Error: {err}</p>}
                        <button type="submit">Submit Test</button>
                    </form>
                </div>
            </div>
        </div>
    );
};

export default CreateTest