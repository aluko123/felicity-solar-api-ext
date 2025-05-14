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
        row.setAttribute('data-id', item.id);

        const idCell = row.insertCell();
        const voltageCell = row.insertCell();
        const percentageCell = row.insertCell();
        const actionsCell = row.insertCell();

        idCell.textContent = item.id;
        voltageCell.textContent = item.voltage;
        percentageCell.textContent = item.percentage;

        
        const editButton = document.createElement('button');
        editButton.textContent = 'Edit'  //set button text
        editButton.onclick = () => openEditModal(item);
        actionsCell.appendChild(editButton);   //append button to the cell
    });
}

// Load all history on page load (optional)
document.addEventListener('DOMContentLoaded', fetchCalibrationHistory);


//modal functions
const editModal = document.getElementById('editModal');
const closeButton = document.querySelector('.close-button');

closeButton.onclick = function() {
    editModal.style.display = 'none';
}

window.onclick = function(event) {
    if (event.target == editModal) {
        editModal.style.display = 'none';
    }
}

function openEditModal(record) {
    document.getElementById('editRecordId').value = record.id;
    document.getElementById('editVoltage').value = record.voltage;
    document.getElementById('editBattery').value = record.percentage;
    editModal.style.display = 'block';
}

async function saveEditedRecord() {
    const id = document.getElementById('editRecordId').value;
    const voltage = parseFloat(document.getElementById('editVoltage').value);
    const percentage = parseInt(document.getElementById('editBattery').value);

    if (percentage > 100 || percentage < 0) {
        alert("Battery percentage must be within range");
        return;
    }

    if (isNaN(voltage) || isNaN(percentage)) {
        alert("Please enter valid numbers for Voltage and Percentage");
        return;
    }
    

    try{
        const response = await fetch(`/api/calibration_data/${id}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({voltage: voltage, percentage: percentage })
        });

        if (!response.ok) {
            const errorData = await response.json();
            console.error("Error updating calibration record:", errorData);
            alert(`Error updating record: ${errorData.error}`);
            return;
        }

        const updatedRecord = await response.json();
        console.log("Record updated successfully:", updatedRecord);
        alert('Record updated succesfully!');
        editModal.style.display = 'none';       //close modal
    
        const tableBody = document.getElementById('calibration-table-body');
        const rows = tableBody.getElementsByTagName('tr');

        console.log('Updated record ID:', updatedRecord.id)
        console.log('Searching for row with data-id:', updatedRecord.id)

        for (let i = 0; i < rows.length; i++) {
            const row = rows[i];
            const rowId = row.getAttribute('data-id');
            console.log('Searching for row with data-id:', rowId)

            if (rowId == updatedRecord.id){
                console.log('Row found!')
                const cells = row.getElementsByTagName('td');
                cells[1].textContent = updatedRecord.voltage;
                cells[2].textContent = updatedRecord.percentage;
                break;
            }
        }
    } catch (error) {
        console.error('Error sending update request:', error);
        alert('Failed to update record. Check console for details.');
    }
}