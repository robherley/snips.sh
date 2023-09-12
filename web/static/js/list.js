const $snipsList = document.getElementById('list-snips')
const snipsList = function ($snipsList) {
    /**
     * Format bytes as human-readable text.
     * 
     * @link https://stackoverflow.com/a/14919494
     * 
     * @param bytes Number of bytes.
     * @param si True to use metric (SI) units, aka powers of 1000. False to use 
     *           binary (IEC), aka powers of 1024.
     * @param dp Number of decimal places to display.
     * 
     * @return Formatted string.
     */
    const humanFileSize = function (bytes, si = false, dp = 1) {
        const thresh = si ? 1000 : 1024;

        if (Math.abs(bytes) < thresh) {
            return bytes + ' B';
        }

        const units = si
            ? ['kB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
            : ['KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];
        let u = -1;
        const r = 10 ** dp;

        do {
            bytes /= thresh;
            ++u;
        } while (Math.round(Math.abs(bytes) * r) / r >= thresh && u < units.length - 1);


        return bytes.toFixed(dp) + ' ' + units[u];
    }

    /**
     * timeAgo
     * https://stackoverflow.com/a/3177838
     * 
     * @param {Date} date 
     * 
     * @returns {string}
     */
    const timeAgo = function (date) {
        console.dir(date)
        var seconds = Math.floor((new Date() - date) / 1000);

        var interval = seconds / 31536000;

        if (interval > 1) {
            return Math.floor(interval) + " years";
        }
        interval = seconds / 2592000;
        if (interval > 1) {
            return Math.floor(interval) + " months";
        }
        interval = seconds / 86400;
        if (interval > 1) {
            return Math.floor(interval) + " days";
        }
        interval = seconds / 3600;
        if (interval > 1) {
            return Math.floor(interval) + " hours";
        }
        interval = seconds / 60;
        if (interval > 1) {
            return Math.floor(interval) + " minutes";
        }

        return Math.floor(seconds) + " seconds";
    }

    const success = function (snips) {
        $snipsList.innerHTML = '';
        if (snips.length === 0) {
            $snipsList.innerHTML = 'No snips found.';
            return
        }

        for (let snip of snips) {
            const snipsListItem = document.createElement('div')
            const updatedAt = new Date(snip.UpdatedAt)

            snipsListItem.innerHTML = `<a href="f/${snip.ID}">${snip.ID}</a> (${snip.Type}) &middot; ${humanFileSize(snip.Size, false, 0)} &middot; ${timeAgo(updatedAt)} ago`
            $snipsList.insertAdjacentElement('beforeend', snipsListItem)
        }
    }

    const failure = function (error) {
        $snipsList.innerHTML = '<div>There was an issue retrieving the latest snips.</div>';
        console.error(error)
    }

    fetch('api/v1/snips', {
        headers: {
            'Content-Type': 'application/json',
            'Accept-Encoding': 'br;q=1.0, gzip;q=0.8, *;q=0.1',
        }
    }).then(d => d.json().then(success)).catch(e => failure(e))
}

if ($snipsList) {
    snipsList($snipsList)
}