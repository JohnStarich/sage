// lifted from https://github.com/esetnik/customize-cra-react-refresh/blob/72f05aa9737d3a1480a5e3f5d84179ee89d4f47a/README.md#installation
const { override } = require("customize-cra");
const { addReactRefresh } = require("customize-cra-react-refresh");

/* config-overrides.js */
module.exports = override(addReactRefresh({ disableRefreshCheck: true }));
