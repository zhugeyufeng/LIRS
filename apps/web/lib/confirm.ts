export function confirmTwice(firstMessage: string, secondMessage: string) {
  return window.confirm(firstMessage) && window.confirm(secondMessage);
}
