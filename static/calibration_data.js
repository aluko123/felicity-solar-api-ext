async function recordCalibration() {
    const voltageInput = document.getElementById('voltage');
    const batteryInput = document.getElementById('battery');

    while (parseInt(batteryInput.value) > 100) {
        alert("Battery percentage cannot be greater than 100");
        batteryInput = document.getElementById('battery');
    }

    const voltage = parseFloat(voltageInput.value);
    const battery = parseInt(batteryInput.value);


    const url = `/api/calibrate_battery`;
    
    const data = {
        voltage: voltage,
        percentage: battery
    }

    try {
        const response = await fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            const errorData = await response.json();
            console.error('Error recording calibration:', errorData);
            alert("Error recording calibration");
            return;
        }

        const responseData = await response.json();
        displayCalibration(responseData)
        console.log('Calibration recorded successfully:', responseData);
        alert("Calibration recorded successfully")
    } catch (error) {
        console.error('There was an error sending calibration data:', error);
    } finally {
        //clear form inputs
        voltageInput.value = '';
        batteryInput.value = '';
    }
}


// async function displayCalibrationData() {
//     const response = await fetch('/api/calibrate_data', {
//         method: 'POST', //use post to trigger the action
//     });

//     if (response.ok) {
//         const data = await response.json();
//         displayHistory(data);
//     } else{
//         const errorData = await response.json();
//         alert(`Error running main function: ${errorData.error}`);
//     }

// }

async function fetchCalibrationHistory() {
    const response = await fetch('/api/calibration_data');
    const data = await response.json();
    displayCalibration(data);
}

function displayCalibration(calibrationData) {
    const tableBody = document.getElementById('calibration-table-body');
    tableBody.innerHTML = ''; // Clear previous data

    calibrationData.forEach(item => {
        const row = tableBody.insertRow();
        row.insertCell().textContent = item.id;
        row.insertCell().textContent = item.voltage;
        row.insertCell().textContent = item.percentage;
        // row.insertCell().textContent = item.timestamp;
    });
}

// Load all history on page load (optional)
document.addEventListener('DOMContentLoaded', fetchCalibrationHistory);