async function runMainFunction() {
    const response = await fetch('/api/run_main', {
        method: 'POST', //use post to trigger the action
    });

    if (response.ok) {
        const data = await response.json();
        displayHistory(data);
    } else{
        const errorData = await response.json();
        alert(`Error running main function: ${errorData.error}`);
    }

}

async function fetchAllHistory() {
    const response = await fetch('/api/history');
    const data = await response.json();
    displayHistory(data);
}


function displayHistory(historyData) {
    const tableBody = document.getElementById('history-table-body');
    tableBody.innerHTML = ''; // Clear previous data

    historyData.forEach(item => {
        const row = tableBody.insertRow();
        row.insertCell().textContent = item.ID;
        row.insertCell().textContent = item.TimeStamp;
        row.insertCell().textContent = item.PvTotalPower;
        row.insertCell().textContent = item.EmsPower;
        row.insertCell().textContent = item.LoadPower;
        row.insertCell().textContent = item.EmsVoltage;
        row.insertCell().textContent = item.BatteryPercentage;
    });
}

// Load all history on page load (optional)
document.addEventListener('DOMContentLoaded', fetchAllHistory);