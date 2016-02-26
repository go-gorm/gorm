var moment = require('moment');


module.exports = {
    // Format a date
    // ex: 'MMMM Do YYYY, h:mm:ss a
    date: function(time, format) {
        return moment(time).format(format);
    },

    // Relative Time
    dateFromNow: function(time) {
        return moment(time).fromNow();
    }
};
