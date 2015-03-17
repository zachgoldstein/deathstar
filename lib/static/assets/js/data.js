
console.log("window.location.hash: " + window.location.hash)

var currentSection = window.location.hash.substring(1, window.location.hash.length);
setSections();

$( "#summary-btn").bind( "click", function(){
    currentSection = "summary"
    setSections();
});

$( "#status-btn").bind( "click", function(){
    currentSection = "status"
    setSections();
});

$( "#options-btn").bind( "click", function(){
    currentSection = "options"
    setSections();
});

$( "#throughput-btn").bind( "click", function(){
    currentSection = "throughput"
    setSections();
});

$( "#latencies-btn").bind( "click", function(){
    currentSection = "latencies"
    setSections();
});

$( "#failures-btn").bind( "click", function(){
    currentSection = "failures"
    setSections();
});

$( "#reqResp-btn").bind( "click", function(){
    currentSection = "reqResp"
    setSections();
});

$( "#raw-btn").bind( "click", function(){
    currentSection = "raw"
    setSections();
});

function setSections(){
    console.log("Setting section to " + currentSection)
    hideAllSections();
    if (currentSection === "summary" || currentSection === "") {
        $("#summary").css("display", "inherit");
    } else if (currentSection === "status") {
        $("#status").css("display", "inherit");
    } else if (currentSection === "options") {
        $("#options").css("display", "inherit");
    } else if (currentSection === "throughput") {
        $("#throughput").css("display", "inherit");
    } else if (currentSection === "latencies") {
        $("#latencies").css("display", "inherit");
    } else if (currentSection === "failures") {
        $("#failures").css("display", "inherit");
    } else if (currentSection === "reqResp") {
        $("#reqResp").css("display", "inherit");
    } else if (currentSection === "raw") {
        $("#raw").css("display", "inherit");
    }
}

function hideAllSections(){
    $("#summary").css("display", "none");
    $("#status").css("display", "none");
    $("#options").css("display", "none");
    $("#throughput").css("display", "none");
    $("#latencies").css("display", "none");
    $("#failures").css("display", "none");
    $("#reqResp").css("display", "none");
    $("#raw").css("display", "none");
}

var socket = io("http://localhost:8081/");

var latestData;
var connected = false;
var googleLoaded = false;

google.load('visualization', '1', {packages: ['corechart']});
google.setOnLoadCallback(doneGoogleLoad);

function doneGoogleLoad() {
    googleLoaded = true;
}

socket.on('connect', function(data){
  console.log('connected');
  connected = true;
  socket.on('disconnect', function(){
    render(latestData);
    console.log('disconnected');
  });
  socket.on('event', function(event){
    try {
        latestData = JSON.parse(event);
        latestData.Latest.RawStats = ""
        latestData.Latest.OverallStats = ""

        render(latestData);
    } catch (e) {
        console.log('could not parse: ',e);
    }
  });
});

function render(latestData) {
    setProgressBar(latestData);

    if (currentSection === "summary") {
        setSummary(latestData);
    } else if (currentSection === "status") {
        setStatus(latestData);
    } else if (currentSection === "options") {
        setOptions(latestData);
    } else if (currentSection === "throughput") {
        setThroughput(latestData);
    } else if (currentSection === "latencies") {
        setLatencies(latestData);
    } else if (currentSection === "failures") {
        buildFailureMap(latestData);
    } else if (currentSection === "reqResp") {

    } else if (currentSection === "raw") {
        $( "#latest").text(JSON.stringify(latestData, null, 2));
    }
}

function setProgressBar(data) {
    if (data.ReqOpts.Mode == "fail") {
        $( "#mode" ).text( "Testing to failure by continually increasing request rate" );
    } else if (data.ReqOpts.Mode == "scale") {
        $( "#mode" ).text( "Testing scale by executing at max request rate" );
    } else if (data.ReqOpts.Mode == "valid") {
        $( "#mode" ).text( "Testing valid responses with a single request" );
    }

    $( "#progress-text" ).text( data.PercentageComplete + "% Time Elapsed" );
    $( "#progressbar" ).css("width",data.PercentageComplete + "%")
    $( "#progressbar" ).attr("aria-valuemax", data.ProgressBarMax)
    $( "#progressbar" ).attr("aria-valuenow", data.ProgressBarCurrent)
    $( "#req-progress-text" ).text( data.ReqBarText);
    $( "#req-progressbar" ).css("width",data.ReqPercentageComplete + "%")
    $( "#req-progressbar" ).attr("aria-valuemax", data.ReqProgressBarMax)
    $( "#req-progressbar" ).attr("aria-valuenow", data.ReqProgressBarCurrent)
}

function setSummary(latestData) {

}

function setOptions(data) {
    $( "#concurrency").text(data.ReqOpts.Concurrency +" workers in pool")
    $( "#cpus").text(data.ReqOpts.CPUs +" CPUs")
    $( "#reqs-to-issue").text(data.ReqOpts.RequestsToIssue + " reqs")
    $( "#render-frequency").text(data.ReqOpts.RenderFrequency/1000000/1000 + "s")
    $( "#analysis-frequency").text(data.ReqOpts.AnalaysisFreqTime/1000000/1000 + "s")
    $( "#keep-alive").text(data.ReqOpts.EnableKeepAlive)
}

function setStatus(data) {
    $("#reqs-issued").text( data.Latest.TotalRequests )
    $("#max-reqs").text(data.ReqOpts.RequestsToIssue)
    $("#time-elapsed").text(data.TimeElapsed)
    $("#max-execution-time").text(data.TotalTime)
    $("#connected").text(connected)
    $("#startTime").text(data.Latest.StartTime)
    if (data.TimeElapsed === data.TotalTime) {
        $("#finished").text("Yes")
    } else {
        $("#finished").text("No")
    }
}

function setThroughput(data) {
    $("#req-rate").text( Math.round(data.Latest.Rate * 100) / 100 + " req/s" )
    $("#throughput-kb").text( data.ThroughputKbs + " kb/s" )
    $("#throughput-resp").text( data.Latest.LatestRespThroughput + " resp/s" )
    $("#mean-throughput-kb").text( Math.round(data.Latest.AverageByteThroughput * 100) / 100  + " kb/s" )
    $("#mean-throughput-resp").text( Math.round(data.Latest.AverageRespThroughput * 100) / 100 + " resp/s" )

    if (!googleLoaded) {
        return
    }

    var throughputResp = new google.visualization.DataTable();
    throughputResp.addColumn('string', 'Time');
    throughputResp.addColumn('number', 'Resp/s');

    var respThroughputs = []
    data.Latest.RespThroughputs.forEach( function (throughput, index) {
        respThroughputs.push([index + "s", throughput ])
    })

    throughputResp.addRows(respThroughputs);

    var options = {
    hAxis: {
      textPosition: 'none',
        gridlines: {
            color: 'transparent'
        },
    },
    vAxis: {
        baselineColor: 'transparent',
        gridlines: {
            color: 'transparent'
        },
    },
    height: 100,
    legend: {position: 'none'},
    }

    var chart = new google.visualization.LineChart( document.getElementById('throughput-resp-chart') );
    chart.draw(throughputResp, options);

    var throughputBytes = new google.visualization.DataTable();
    throughputBytes.addColumn('string', 'Time');
    throughputBytes.addColumn('number', 'kb/s');

    var kbThroughputs = []
    data.Latest.ByteThroughputs.forEach( function (throughput, index) {
        kbThroughputs.push([index + "", throughput/1000 ])
    })
    throughputBytes.addRows(kbThroughputs);

    var options = {
    hAxis: {
      textPosition: 'none',
        gridlines: {
            color: 'transparent'
        },
    },
    vAxis: {
        baselineColor: 'transparent',
        gridlines: {
            color: 'transparent'
        },
    },
    height: 100,
    legend: {position: 'none'},
    }

    var chart = new google.visualization.LineChart( document.getElementById('throughput-kb-chart') );
    chart.draw(throughputBytes, options);
}

function setLatencies(data) {

    $("#max-response-time").text(data.MaxResponseTime + "s")
    $("#min-response-time").text(data.MinResponseTime + "s")
    $("#mean-response-time").text(data.AvgResponseTime + "s")
    $("#top-percentile-time").text(data.TopPercentileTime + "s")
    $("#top-percentile-title").text(data.TopPercentileTimeTitle + " Response Time")

    if (!googleLoaded) {
        return
    }

    var percentiles = []

    data.Latest.Percentiles.forEach( function (percentile, index) {
        percentiles.push([percentile * 100 + "", data.LatestTotalPercentiles[index]])
    })

    var options = {
        hAxis: {
          title: 'Percentiles'
        },
        vAxis: {
          title: 'Latency(s)'
        },
        legend: {position: 'none'},
    };

    var totalPercentiles = new google.visualization.DataTable();
    totalPercentiles.addColumn('string', 'Total Percentiles');
    totalPercentiles.addColumn('number', 'Latency (s)');

    totalPercentiles.addRows(percentiles);

    var chart = new google.visualization.LineChart( document.getElementById('total-latency-chart') );
    chart.draw(totalPercentiles, options);

    var percentiles = []

    data.Latest.Percentiles.forEach( function (percentile, index) {
        percentiles.push([percentile * 100 + "", data.LatestConnectPercentiles[index]])
    })

    var options = {
        hAxis: {
          title: 'Percentiles'
        },
        vAxis: {
          title: 'Latency(s)'
        },
        legend: {position: 'none'},
    };

    var totalPercentiles = new google.visualization.DataTable();
    totalPercentiles.addColumn('string', 'Total Percentiles');
    totalPercentiles.addColumn('number', 'Latency (s)');

    totalPercentiles.addRows(percentiles);

    var chart = new google.visualization.LineChart( document.getElementById('connect-latency-chart') );
    chart.draw(totalPercentiles, options);

      var options = {
        hAxis: {
          title: 'Latency(s)'
        },
        vAxis: {
          title: 'Count'
        },
        chartArea: {
           height: '40%'
        },
        legend: {position: 'none'},
      };

      var totalHistData = [['Latencies']]
      data.TotalHistogramTimes.forEach( function (histTime) {
        totalHistData.push([histTime])
      })


      var histData = google.visualization.arrayToDataTable(totalHistData);

        var chart = new google.visualization.Histogram(document.getElementById('total-latency-histogram'));
        chart.draw(histData, options);

      var options = {
        hAxis: {
          title: 'Latency(s)'
        },
        vAxis: {
          title: 'Count'
        },
        chartArea: {
           height: '40%'
        },
        legend: {position: 'none'},
      };

      var connectHistData = [['Latencies']]
      data.ConnectionHistogramTimes.forEach(function(histTime){
        connectHistData.push([histTime])
      })

      var histData = google.visualization.arrayToDataTable(connectHistData);

        var chart = new google.visualization.Histogram(document.getElementById('connect-latency-histogram'));
        chart.draw(histData, options);
}

function buildFailureMap(data){
    $("#failuresTitle").text("Failures", data.Latest.Failures)
    var tbody = $("#failureTable").html("")
    for (var failureDescription in data.FailureMap) {
        var row = $("<tr></tr>");
        row.append( $("<td></td>").html(data.FailureMap[failureDescription]) );
        row.append( $("<td></td>").html(failureDescription) );
        tbody.append(row);
    }
}

