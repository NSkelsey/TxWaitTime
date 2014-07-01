function AvgTime() {
// Creates a series of html widgets with avg conf time
// Also adds a porportion chart to confirmation time
    var width = 250,
    height = 150
    maxRad = (height)/2 - 10;

    function builder(selection, summary_data) {
        // filters out data with negative wait times and sorts by size

        summary_data = summary_data.sort(function(a, b) {
            return b.confirmed - a.confirmed
        });

        summary_data = summary_data.filter(function(d) {
            if (d.confirmations < 1) { return false }
            return d.avg > 0;
        });

        var divEnter = selection.selectAll("div")
            .data(summary_data)
          .enter().append("div");

        divEnter.attr("class", "col-md-4 sum-entry");

        function fixNames(d) {
            if (d.kind in nameMap) {
                return nameMap[d.kind]
            }
            return d.kind
        }

        divEnter.append("a")
             .attr("href", function(d){return "#" + d.kind})
          .append("h2")
             .attr("class", "kind")
             .text(fixNames)
        

        function formatAvg(d) {
            secs = d.avg;
            
            mins = Math.floor(secs/60);
            secs = String(Math.floor(secs % 60));

            if (secs.length == 1) {
                secs = secs
            }
            return mins + "m " + secs + "s";
        }

        divEnter.append("b")
            .attr("class", "time")
            .text(formatAvg);   


        // relative size circle

        var radScale = d3.scale.log()
           .domain([1, d3.max(summary_data, function(d){return d.confirmed})])
           .range([5, maxRad]);

        // http://bl.ocks.org/mbostock/3887235
        var colorCat = d3.scale.category20()
            .domain(d3.map(avg_conf_time, function(d){return d.kind}))


        var sizeChart = divEnter.append("svg")
            .style("width", width)
            .style("height", height)

        sizeChart.append("circle")
            .attr("r", function(d){ 
                    return radScale(d.confirmed)}
                  )
            .attr("cx", width/2)
            .attr("cy", height/2)
            .style("fill", function(d) { return colorCat(d.kind) })

        sizeChart.append("text")
            .attr("x", width/2)
            .attr("y", height/2)
            .text(function(d) { return d.confirmed });

    }

    return builder
}


function Histogram() {
    var margin = {top: 20, right: 30, bottom: 30, left: 50},
        width = 400 - margin.left - margin.right,
        height = 300 - margin.top - margin.bottom;

    function builder(selection, bucket_data){

        var chart = selection.append("svg")
               .attr('width', width + margin.left + margin.right)
               .attr('height', height + margin.top + margin.bottom)
             .append("g")
               .attr("transform", "translate(" + margin.left + "," + margin.top + ")");


        var maxX = d3.max(bucket_data, function(d){return d.col}),
            maxY = d3.max(bucket_data, function(d){return d.count});

        var y = d3.scale.linear()
            .domain([0, maxY])
            .range([height, 0]);

        var x = d3.scale.linear()
            .domain([0, maxX])
            .range([0, width]);

        var barWidth = width / maxX;

        // attach data
        var bar = chart.selectAll("g")
            .data(bucket_data)
          .enter().append("g")
            .attr("transform", function(d) { 
                        var x = (d.col * barWidth)
                        return "translate(" + x + ",0)"; 
                        
                        });

        bar.append("rect")
            .attr("y", function(d) {return y(d.count)})
            .attr("height", function(d) { return height - y(d.count) })
            .attr("width", barWidth - 2);

        // attach labels
        var yAxis = d3.svg.axis()
            .scale(y)
            .orient("left");

        var xAxis = d3.svg.axis()
            .scale(x)
            .orient("bottom");

        chart.append("g")
            .attr("class", "y axis")
            .call(yAxis);

        chart.append("g")
            .attr("class", "x axis")
            .attr("transform", "translate(0," + height + ")")
            .call(xAxis);
    }
    return builder
}


function PieChart() {
    var labelMap = {
    
                    "forgotten" : "Forgotten",
                    "unseen"    : "Unseen",
                    "total"     : "Total",
                    "mempool"   : "Unconfirmed",
                    "confirmed" : "Confirmed",
    };
    

    hex = ['rgb(215,25,28)','rgb(166,217,106)','#e3e3e3','rgb(253,174,97)','rgb(26,150,65)'];

    var margin = {top: 10, right: 0, bottom: 10, left: 10},
        width  = 300 - margin.right - margin.left,
        height = 250 - margin.top - margin.bottom,
        radius = Math.min(width, height) / 2,
        color  = d3.scale.ordinal()
            .domain(Object.keys(labelMap))
            .range(hex);


    function builder(selection, conf_rates) {
        var chart = selection.append("div")
            .attr("class", "pie-chart")

        var rates = [];
        for (key in conf_rates) {
            var num = Number(conf_rates[key]);
            if (num < 1) {
                continue;
            }
            var obj = { 
                num: num,
                bucket: key,
            };
            rates.push(obj);
        }

        var total = rates.reduce(function(acc, d){return acc + d.num}, 0);

        var arc = d3.svg.arc()
            .outerRadius(radius)
            .innerRadius(0);
    
        var pie = d3.layout.pie()
            .sort(null)
            .value(function(d) { return d.num; });


        /* The pie chart itself */
        var svg = chart.append("svg")
            .attr("width", width + margin.right + margin.left)
            .attr("height", height + margin.top + margin.bottom)
          .append("g")
            .attr("transform", function(_){
                var xOff = width / 2 + margin.left;
                var yOff = height / 2 + margin.top;
                return "translate(" +  xOff + "," + yOff + ")"
            });

        var g = svg.selectAll(".arc")
            .data(pie(rates))
          .enter().append("g")
            .attr("class", "arc");
    
        g.append("path")
            .attr("d", arc)
            .style("fill", function(a) { return color(a.data.bucket); });
        

        /* The key table */
        var table = chart.append("table")
            .attr("class", "pie-table")
          .append("tbody");

        var tableData = rates.slice(0);
        tableData.push({bucket: "total", num: total});

        var trows = table.selectAll("tr")
          .data(tableData).enter().append("tr")

        // Shooting myself in the foot.
        var fmap = [
            function(d) {
                var fill = color(d.bucket);
                return '<svg class="box" width="15" height="15" ><rect x="0" y="0" width="15" height="15" fill="' + fill + '" /></svg>';
                },
            function(d) {
                return labelMap[d.bucket];
            },
            function(d) {
                return d.num;
            },
            function(d) {
                return d3.format(".2f")(100* (d.num / total) ) + "%";
            }
        ];
        
        // Evaluates the row column functions.
        function colMap(d) {
            ret = fmap.map(function(f){return f(d);});
            return ret;
        }

        var tds = trows.selectAll("td")
            .data(colMap)
          .enter().append("td")
            .html(function(d) { 
                return d
            });
            

    }
    return builder
}
