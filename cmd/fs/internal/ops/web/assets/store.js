// store.js 提供最小状态容器，管理页面共享状态与订阅。
export function createStore(initialState) {
  let state = { ...initialState };
  const listeners = new Set();

  function setState(partial) {
    state = { ...state, ...partial };
    listeners.forEach((listener) => listener(state));
  }

  function getState() {
    return state;
  }

  function subscribe(listener) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  }

  return {
    setState,
    getState,
    subscribe,
  };
}
