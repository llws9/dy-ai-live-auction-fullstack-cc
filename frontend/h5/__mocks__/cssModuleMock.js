const proxy = new Proxy(
  {},
  {
    get: (target, prop) => {
      if (prop === '__esModule') return true;
      if (prop === Symbol.toPrimitive) return () => 'default';
      if (prop === 'toString') return () => 'default';
      if (prop in target) return target[prop];
      return String(prop);
    },
  },
);

module.exports = proxy;
module.exports.default = proxy;
