type ConfirmHandler = (message: string) => boolean;

let confirmHandler: ConfirmHandler | null = null;

export function confirmTwice(firstMessage: string, secondMessage: string) {
  const confirm = confirmHandler ?? ((message: string) => window.confirm(message));
  return confirm(firstMessage) && confirm(secondMessage);
}

export function setConfirmHandlerForTests(handler: ConfirmHandler | null) {
  confirmHandler = handler;
}
