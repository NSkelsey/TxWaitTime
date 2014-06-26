function AvgTime() {
// Creates a series of html widgets with avg conf time
// Also adds a porportion chart to confirmation time
    var nameMap = {
       pubkeyhash : "Standard",
       nulldata   : "Null Data",
       scripthash : "Script Hash",
       pubkey     : "Public Key",
       multisig   : "Multisignature",
               },
    width = 250,
    height = 150
    maxRad = (height)/2 - 10;

    function mapper(selection, avg_conf_time) {

        avg_conf_time = avg_conf_time.sort(function(a, b) {
            return b.confirmed - a.confirmed
        });

        avg_conf_time = avg_conf_time.filter(function(d) {
            if (d.confirmations < 1) { return false }
            return d.avg > 0;
        });

        var divEnter = selection.selectAll("div")
            .data(avg_conf_time)
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
           .domain([1, d3.max(avg_conf_time, function(d){return d.confirmed})])
           .range([5, maxRad]);

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

    return mapper
}
