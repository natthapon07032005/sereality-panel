axios.defaults.headers.post['Content-Type'] = 'application/x-www-form-urlencoded; charset=UTF-8';
axios.defaults.headers.common['X-Requested-With'] = 'XMLHttpRequest';

axios.interceptors.request.use(
    (config) => {
        if (config.data instanceof FormData) {
            config.headers['Content-Type'] = 'multipart/form-data';
        } else if (!isJsonRequest(config)) {
            config.data = Qs.stringify(config.data, {
                arrayFormat: 'repeat',
            });
        }
        return config;
    },
    (error) => Promise.reject(error),
);

function isJsonRequest(config) {
    const headers = config.headers;
    if (!headers) {
        return false;
    }
    const contentType = typeof headers.get === 'function'
        ? headers.get('Content-Type')
        : headers['Content-Type'] || headers['content-type'];
    return typeof contentType === 'string'
        && contentType.split(';', 1)[0].trim().toLowerCase() === 'application/json';
}

axios.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response) {
            const statusCode = error.response.status;
            // Check the status code
            if (statusCode === 401) { // Unauthorized
                return window.location.reload();
            }
        }
        return Promise.reject(error);
    }
);
