import http from 'k6/http';
import {check, sleep} from 'k6';

export let options = {
    thresholds: {
        http_req_failed: ['rate<0.01'], // http errors should be less than 1%
        http_req_duration: ['p(95)<200'], // 95% of requests should be below 200ms
    },
    stages: [
        { duration: '30s', target: 50 }, // Boost to 50 users for the time period of N
        { duration: '1m', target: 50 },  // Lock 50 users for the time period of N
        { duration: '30s', target: 0 },  // Drop users rate to 0 slowly for the time period of N
    ],
};

export default function () {
    let res = http.get('http://localhost:9000/users/0/100');
    check(res, {
        'status is 200': (r) => r.status === 200,
    });
    sleep(1);
}