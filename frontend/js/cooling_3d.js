import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

var _state = {
  scene: null,
  camera: null,
  renderer: null,
  controls: null,
  cubes: [],
  labelsContainer: null,
  raycaster: null,
  mouse: null,
  animationId: null,
  devices: null,
  onDeviceClick: null,
  _boundClick: null,
  _boundResize: null
};

function copColor(cop) {
  if (cop > 6) return 0x00ff88;
  if (cop >= 4) return 0xffcc00;
  return 0xff4455;
}

function addCubes(arr, rows, cols, sx, sy, sz, offX, offY, offZ) {
  var idx = 0;
  for (var r = 0; r < rows; r++) {
    for (var c = 0; c < cols; c++) {
      if (idx >= arr.length) break;
      var dev = arr[idx];
      var geom = new THREE.BoxGeometry(sx, sy, sz);
      var col = copColor(dev.cop);
      var mat = new THREE.MeshPhongMaterial({ color: col, transparent: true, opacity: 0.88, emissive: col, emissiveIntensity: 0.15 });
      var mesh = new THREE.Mesh(geom, mat);
      var px = offX + c * (sx + 1.5);
      var pz = offZ + r * (sz + 1.5);
      mesh.position.set(px, offY + sy / 2, pz);
      mesh.userData = { deviceCode: dev.code, deviceId: dev.id };
      if (dev.alert) mesh.userData.pulse = true;
      _state.scene.add(mesh);
      _state.cubes.push(mesh);
      idx++;
    }
  }
}

function addPipeline(x1, y1, z1, x2, y2, z2, color) {
  var points = [];
  if (y1 !== y2) {
    points.push(new THREE.Vector3(x1, y1, z1));
    points.push(new THREE.Vector3(x1, (y1 + y2) / 2, z1));
    points.push(new THREE.Vector3(x2, (y1 + y2) / 2, z2));
    points.push(new THREE.Vector3(x2, y2, z2));
  } else {
    points.push(new THREE.Vector3(x1, y1, z1));
    points.push(new THREE.Vector3(x2, y2, z2));
  }
  var curve = new THREE.CatmullRomCurve3(points);
  var tubeGeom = new THREE.TubeGeometry(curve, 32, 0.08, 6, false);
  var tubeMat = new THREE.MeshPhongMaterial({ color: color, transparent: true, opacity: 0.5 });
  _state.scene.add(new THREE.Mesh(tubeGeom, tubeMat));
}

function updateLabels() {
  var labelsContainer = _state.labelsContainer;
  if (!labelsContainer) return;
  labelsContainer.innerHTML = '';
  var w = _state.renderer.domElement.clientWidth;
  var h = _state.renderer.domElement.clientHeight;
  _state.cubes.forEach(function(cube) {
    var pos = cube.position.clone();
    pos.y += 1.5;
    pos.project(_state.camera);
    if (pos.z > 1) return;
    var x = (pos.x * 0.5 + 0.5) * w;
    var y = (-pos.y * 0.5 + 0.5) * h;
    var label = document.createElement('div');
    label.className = 'three-label';
    label.textContent = cube.userData.deviceCode;
    label.style.left = x + 'px';
    label.style.top = y + 'px';
    labelsContainer.appendChild(label);
  });
}

function animate() {
  _state.animationId = requestAnimationFrame(animate);
  _state.controls.update();
  var time = Date.now() * 0.003;
  _state.cubes.forEach(function(cube) {
    if (cube.userData.pulse) {
      cube.material.emissiveIntensity = 0.15 + 0.2 * Math.sin(time + cube.position.x);
    }
  });
  updateLabels();
  _state.renderer.render(_state.scene, _state.camera);
}

function handleClick(event) {
  var rect = _state.renderer.domElement.getBoundingClientRect();
  _state.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
  _state.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;
  _state.raycaster.setFromCamera(_state.mouse, _state.camera);
  var intersects = _state.raycaster.intersectObjects(_state.cubes);
  if (intersects.length > 0) {
    var devCode = intersects[0].object.userData.deviceCode;
    if (_state.onDeviceClick) _state.onDeviceClick(devCode);
  }
}

export function init(containerId, labelsId, devices, onDeviceClick) {
  var container = document.getElementById(containerId);
  var w = container.clientWidth;
  var h = container.clientHeight;

  _state.devices = devices;
  _state.onDeviceClick = onDeviceClick;
  _state.labelsContainer = document.getElementById(labelsId);
  _state.raycaster = new THREE.Raycaster();
  _state.mouse = new THREE.Vector2();

  _state.scene = new THREE.Scene();
  _state.scene.background = new THREE.Color(0x0a0e1a);
  _state.scene.fog = new THREE.Fog(0x0a0e1a, 60, 120);

  _state.camera = new THREE.PerspectiveCamera(50, w / h, 0.1, 500);
  _state.camera.position.set(30, 35, 50);
  _state.camera.lookAt(0, 0, 0);

  _state.renderer = new THREE.WebGLRenderer({ antialias: true });
  _state.renderer.setSize(w, h);
  _state.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
  container.appendChild(_state.renderer.domElement);

  _state.controls = new OrbitControls(_state.camera, _state.renderer.domElement);
  _state.controls.enableDamping = true;
  _state.controls.dampingFactor = 0.05;
  _state.controls.maxPolarAngle = Math.PI / 2.1;

  var ambient = new THREE.AmbientLight(0x334466, 1.5);
  _state.scene.add(ambient);
  var dir = new THREE.DirectionalLight(0xffffff, 1.2);
  dir.position.set(20, 40, 30);
  _state.scene.add(dir);

  var gridHelper = new THREE.GridHelper(80, 40, 0x1a2540, 0x111828);
  _state.scene.add(gridHelper);

  var chillers = devices.filter(function(d) { return d.type === 'chiller'; });
  var towers = devices.filter(function(d) { return d.type === 'cooling_tower'; });
  var pacs = devices.filter(function(d) { return d.type === 'precision_ac'; });
  var cdus = devices.filter(function(d) { return d.type === 'cdu'; });

  _state.cubes = [];

  addCubes(chillers, 2, 4, 2.5, 2.5, 2.5, -9, 0, -4);
  addCubes(towers, 3, 4, 2, 2, 2, -8.5, 0, -16);
  addCubes(pacs, 8, 10, 0.8, 0.8, 0.8, -16, 0, 5);
  addCubes(cdus, 4, 5, 1, 1, 1, -8, 0, 16);

  for (var i = 0; i < chillers.length; i++) {
    var cr = Math.floor(i / 4), cc = i % 4;
    var cx = -9 + cc * 4;
    var cz = -4 + cr * 4;
    var ti = i < towers.length ? i : towers.length - 1;
    var tr = Math.floor(ti / 4), tc = ti % 4;
    var tx = -8.5 + tc * 3.5;
    var tz = -16 + tr * 3.5;
    addPipeline(cx, 2.5, cz, tx, 2, tz, 0x4488ff);
  }

  for (var i = 0; i < chillers.length; i++) {
    var cr = Math.floor(i / 4), cc = i % 4;
    var cx = -9 + cc * 4;
    var cz = -4 + cr * 4;
    for (var j = 0; j < 5; j++) {
      var pi = i * 5 + j;
      if (pi < pacs.length) {
        var pr = Math.floor(pi / 10), pc = pi % 10;
        var px = -16 + pc * 3.2;
        var pz = 5 + pr * 1.8;
        addPipeline(cx, 1.25, cz + 1.25, px, 0.4, pz, 0x00ccaa);
      }
    }
  }

  for (var i = 0; i < chillers.length; i++) {
    var cr = Math.floor(i / 4), cc = i % 4;
    var cx = -9 + cc * 4;
    var cz = -4 + cr * 4;
    for (var j = 0; j < 3; j++) {
      var ci = i * 3 + j;
      if (ci < cdus.length) {
        var cduR = Math.floor(ci / 5), cduC = ci % 5;
        var cduX = -8 + cduC * 3;
        var cduZ = 16 + cduR * 2.5;
        addPipeline(cx, 1.25, cz + 1.25, cduX, 0.5, cduZ, 0x8866ff);
      }
    }
  }

  _state._boundClick = handleClick;
  _state._boundResize = resize;
  _state.renderer.domElement.addEventListener('click', _state._boundClick);
  window.addEventListener('resize', _state._boundResize);

  animate();
}

export function updateDevices(devices) {
  _state.devices = devices;
  devices.forEach(function(dev) {
    var cube = _state.cubes.find(function(c) { return c.userData.deviceCode === dev.code; });
    if (cube) {
      cube.material.color.setHex(copColor(dev.cop));
      cube.material.emissive.setHex(copColor(dev.cop));
      cube.userData.pulse = !!dev.alert;
    }
  });
}

export function resize() {
  var container = _state.renderer.domElement.parentElement;
  if (!container) return;
  var w = container.clientWidth;
  var h = container.clientHeight;
  _state.camera.aspect = w / h;
  _state.camera.updateProjectionMatrix();
  _state.renderer.setSize(w, h);
}

export function destroy() {
  if (_state.animationId) {
    cancelAnimationFrame(_state.animationId);
    _state.animationId = null;
  }
  if (_state._boundClick) {
    _state.renderer.domElement.removeEventListener('click', _state._boundClick);
  }
  if (_state._boundResize) {
    window.removeEventListener('resize', _state._boundResize);
  }
  _state.cubes.forEach(function(cube) {
    cube.geometry.dispose();
    cube.material.dispose();
  });
  _state.scene.traverse(function(obj) {
    if (obj.geometry) obj.geometry.dispose();
    if (obj.material) {
      if (Array.isArray(obj.material)) {
        obj.material.forEach(function(m) { m.dispose(); });
      } else {
        obj.material.dispose();
      }
    }
  });
  if (_state.renderer) {
    _state.renderer.dispose();
    if (_state.renderer.domElement && _state.renderer.domElement.parentNode) {
      _state.renderer.domElement.parentNode.removeChild(_state.renderer.domElement);
    }
  }
  _state.scene = null;
  _state.camera = null;
  _state.renderer = null;
  _state.controls = null;
  _state.cubes = [];
  _state.labelsContainer = null;
  _state.raycaster = null;
  _state.mouse = null;
  _state.devices = null;
  _state.onDeviceClick = null;
  _state._boundClick = null;
  _state._boundResize = null;
}

export function getCubes() {
  return _state.cubes;
}
