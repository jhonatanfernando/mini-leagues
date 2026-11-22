function handler(event) {
  var request = event.request;
  if (request.uri === '/auth/callback') {
    request.uri = '/index.html';
  }
  return request;
}
