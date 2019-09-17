// Electron entrypoint
const {
  BrowserWindow,
  Menu,
  app,
  shell,
} = require('electron');
const { execFile } = require('child_process');
const path = require('path');
const fs = require('fs');

// port = (s * a * g * e) % 2^16
const SagePort = 46745

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (require('electron-squirrel-startup')) { // eslint-disable-line global-require
  app.quit();
}

// All user data is stored here
const DataDirectory = path.join(app.getPath('userData'), "data")
const LedgerFile = path.join(DataDirectory, "ledger.journal")
const RulesFile = path.join(DataDirectory, "ledger.rules")

// Keep a global reference of the window object, if you don't, the window will
// be closed automatically when the JavaScript object is garbage collected.
let mainWindow;
let sageServer;

const createWindow = () => {
  // Create the browser window.
  mainWindow = new BrowserWindow({
    width: 800,
    height: 600,
  });

  // and load the index.html of the app.
  mainWindow.loadURL(`http://localhost:${SagePort}`);

  // Open the DevTools.
  //mainWindow.webContents.openDevTools();
  
  // Emitted when the window is closed.
  mainWindow.on('closed', () => {
    // Dereference the window object, usually you would store windows
    // in an array if your app supports multi windows, this is the time
    // when you should delete the corresponding element.
    mainWindow = null;
  });

  const defaultMenuItems = Menu.getApplicationMenu().items;
  const firstMenus = [], remainingMenus = [];
  let foundSplitMenu = false;
  for (item of defaultMenuItems) {
    if (! foundSplitMenu) {
      firstMenus.push(item)
      foundSplitMenu = item.label === 'View'
    } else {
      remainingMenus.push(item)
    }
  }
  const newMenu = [].concat(
    firstMenus,
    [
      {
        label: 'Data',
        submenu: [
          {
            label: 'Show data folder',
            click: () => shell.showItemInFolder(LedgerFile),
          },
        ],
      },
    ],
    remainingMenus,
  );
  Menu.setApplicationMenu(Menu.buildFromTemplate(newMenu))
};

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on('ready', createWindow);

// Quit when all windows are closed.
app.on('window-all-closed', () => {
  // On OS X it is common for applications and their menu bar
  // to stay active until the user quits explicitly with Cmd + Q
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  // On OS X it's common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  if (mainWindow === null) {
    createWindow();
  }
});

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and import them here.
let sageServerName = "sage-server"
if (! app.isPackaged) {
  sageServerName = path.join("out", "sage")
}

let executable = path.join(app.getAppPath(), "..", sageServerName)
if (process.platform === 'win32') {
  executable += ".exe"
}
fs.mkdirSync(DataDirectory, {recursive: true})

sageServer = execFile(executable, [
  '-server', '-port', SagePort,
  '-data', DataDirectory,
  '-ledger', LedgerFile,
  '-rules', RulesFile,
], function(err) {
  if (err === null) {
    return
  }
  app.quit()
  throw Error(`Failed to run ${executable}: ${err}`)
})

sageServer.stdout.pipe(process.stdout)
sageServer.stderr.pipe(process.stderr)

app.on('quit', () => {
  sageServer.kill('SIGINT')
})
