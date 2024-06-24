$(document).ready(function() {
    $('data-placeholder').addClass('d-none');

    $table = $('#data-table');
    $tbody = $table.find('tbody');

    const token = setInterval(function() {

        $.get('/admin-data', function(data) {
            console.log(data);
            $tbody.empty();

            $tbody.append(`
                <tr>
                    <td>${data.Name}</td>
                    <td>${data.Address}</td>
                    <td>${data.Health}</td>
                    <td>${data.State}</td>
                    <td>${data.NodesAlive}</td>
                    <td>
                        Heap Size: ${data.MemStats.Alloc} MB <br>
                        Total Heap Increment: ${data.MemStats.TotalAlloc} MB <br>
                        Currently Used: ${data.MemStats.Sys} MB <br>
                    </td>
                    <td>${data.CPUs}</td>
                    <td>${data.GoRoutines}</td> 
                </tr>
             `);

            if (data.Others) {
                data.Others.forEach(function(other) {
                    $tbody.append(`
                        <tr>
                            <td>${other.Name}</td>
                            <td>${other.Address}</td>
                            <td>${other.Health}</td>
                            <td>${other.State}</td>
                            <td>${other.NodesAlive}</td>
                            <td>
                                Currently Used: ${other.MemStats.Sys} MB <br>
                                <small>Heap Size: ${other.MemStats.Alloc} MB <br>
                                All Heap Inc.: ${other.MemStats.TotalAlloc} MB <br></small>
                            </td>
                            <td>${other.CPUs}</td>
                            <td>${other.GoRoutines}</td> 
                        </tr>
                    `);
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