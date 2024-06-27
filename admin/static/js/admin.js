function renderTableRow(data) {
    return `
                <tr>
                    <td>${data.Name}</td>
                    <td>${data.Address}</td>
                    <td>${data.Health}</td>
                    <td>${data.State}</td>
                    <td>${data.NodesAlive}</td>
                    <td>
                        Heap Size: ${data.MemStats.Alloc} MB <br>
                        <small>Total Heap Increment: ${data.MemStats.TotalAlloc} MB <br>
                        Currently Used: ${data.MemStats.Sys} MB <br></small>
                    </td>
                    <td>${data.CPUs}</td>
                    <td>${data.GoRoutines}</td> 
                    <td>${data.QueueCount}</td>
                </tr>
             `;
}

$(document).ready(function() {
    $('.data-placeholder').addClass('d-none');

    $table = $('#data-table');
    $tbody = $table.find('tbody');

    const token = setInterval(function() {

        $.get('/admin-data', function(data) {
            console.log(data);
            $tbody.empty();

            $tbody.append(renderTableRow(data));

            if (data.Others) {
                data.Others.forEach(function(other) {
                    $tbody.append(renderTableRow(other));
                });
            }

            $table.removeClass('d-none');
        })
    }, 1000);
})

/*

		data := Data{
			NodesAlive: o.mList.NumMembers(),
			MemStats: MemStats{
				Alloc:      (memStats.Alloc / 1024) / 1024,
				TotalAlloc: (memStats.TotalAlloc / 1024) / 1024,
				Sys:        (memStats.Sys / 1024) / 1024,
			},
			CPUs:       runtime.NumCPU(),
			GoRoutines: runtime.NumGoroutine(),
			Health:     o.mList.GetHealthScore(),
			State:      stateToString(o.mList.LocalNode().State),
		}


                                <th>
                                    Memory: Current Heap Size
                                </th>
                                <th>
                                    Memory: Total Heap Increment
                                </th>
                                <th>
                                    Memory: Currently Used (approx)
                                </th>
 */