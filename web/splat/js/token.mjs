import { adjacentCells, cellPosition, normalizeTokenPath, positionCell, terrainHeight, tokenCell, validTokenCell } from "./occupied_cells.mjs";
import { createOccupiedCells } from "./token_cells.mjs";
import { createTokenSprite, hasClips, roleForToken } from "./token_sprite.mjs";

function role(token) { return roleForToken(token); }
function same(a,b) { return a.c === b.c && a.r === b.r; }
function gone(token) { return [token.status,...(Array.isArray(token.statuses) ? token.statuses : [])].some(value => ["removed","fled","defeated"].includes(String(value).toLowerCase())); }
function paceSeconds(token) {
  const milliseconds = Number(token?.step_ms);
  return Number.isFinite(milliseconds) && milliseconds > 0 ? milliseconds / 1000 : .25;
}
function place(entry,position) {
  entry.position = position.slice();
  entry.entity.setPosition(position[0],position[1]+entry.height/2,position[2]);
}
function cameraXY(camera) {
  if (!camera || typeof camera.getPosition !== "function") return null;
  const position = camera.getPosition();
  if (!position || !Number.isFinite(position.x) || !Number.isFinite(position.z)) return null;
  return [position.x,position.z];
}
function routeFor(grid, entry, token, path) {
  if (!token.path?.length) return [];
  const current = positionCell(grid,entry.position);
  const route = path.slice();
  if (same(current,route[0])) route.shift();
  if (!route.length) return [];
  if (!adjacentCells(current,route[0])) return null;
  return route.map(cell => cellPosition(grid,cell));
}
class TokenController {
  constructor({pc,app,grid,layer,spriteLayer,camera,effects,reducedMotion=false,onState=()=>{}}) {
    Object.assign(this,{pc,app,grid,layer,spriteLayer: spriteLayer ?? pc?.LAYERID_WORLD,camera,effects,reducedMotion,onState});
    this.entries = new Map(); this.sceneSeq = -1; this.followId = null;
    this.enabled = true; this.gridVisible = true; this.paused = false; this.destroyed = false;
    this.lastCameraXY = null; this.occupiedSignature = null;
    this.cells = createOccupiedCells({pc,app,grid,layer});
  }
  remove(id) {
    const entry = this.entries.get(id);
    if (!entry) return;
    entry.sprite.destroy(); this.entries.delete(id);
    if (this.followId === id) this.clearFollow();
  }
  acceptToken(token) {
    let entry = this.entries.get(token.id);
    if (gone(token)) { this.remove(token.id); return; }
    const seq = Number(token.anim_seq ?? 0);
    if (!Number.isSafeInteger(seq) || seq < 0 || !validTokenCell(this.grid,token.cell)) return;
    if (entry && seq <= entry.seq) { entry.token = {...entry.token,name:token.name ?? entry.token.name,clips:token.clips ?? entry.token.clips}; this.adoptClips(entry,token,false); return; }
    if (entry && role(entry.token) !== role(token)) { this.remove(token.id); entry=null; }
    const path = normalizeTokenPath(this.grid,token);
    if (!path) return;
    const signature = JSON.stringify([tokenCell(token.cell),path]);
    if (entry?.signature === signature) { entry.seq=seq; entry.token=token; return; }
    const route = entry ? routeFor(this.grid,entry,token,path) : [];
    if (route === null) return;
    if (!entry) {
      entry = {sprite:createTokenSprite({pc:this.pc,app:this.app,token,layer:this.spriteLayer}), token,seq,queue:[],elapsed:0};
      entry.entity=entry.sprite.entity; entry.material=entry.sprite.material; entry.height=entry.sprite.height;
      this.entries.set(token.id,entry);
    }
    this.adoptClips(entry,token,true);
    entry.token=token; entry.seq=seq; entry.signature=signature;
    entry.queue=route; entry.elapsed=0; entry.start=entry.position?.slice();
    entry.entity.enabled=this.enabled;
    if (!route.length || this.reducedMotion) this.snap(entry);
    entry.sprite.faceCamera(this.camera);
  }
  apply(snapshot={}) {
    const seq = Number(snapshot.seq ?? 0);
    if (this.destroyed || !Number.isSafeInteger(seq) || seq <= this.sceneSeq) return false;
    this.sceneSeq=seq;
    const seen = new Set();
    for (const token of (Array.isArray(snapshot.tokens) ? snapshot.tokens : []).slice(0,64)) {
      if (typeof token?.id !== "string" || !token.id || seen.has(token.id)) continue;
      seen.add(token.id); this.acceptToken(token);
    }
    for (const id of this.entries.keys()) if (!seen.has(id)) this.remove(id);
    this.refresh(); return true;
  }
  /** adoptClips swaps between the stand-in and the green-screen loop when
   * clips arrive or go, and restarts a one-shot on a new anim_seq. */
  adoptClips(entry,token,newSeq) {
    if (Boolean(entry.sprite.video) !== hasClips(token)) {
      const position = entry.position?.slice();
      entry.sprite.destroy();
      entry.sprite=createTokenSprite({pc:this.pc,app:this.app,token,layer:this.spriteLayer});
      entry.entity=entry.sprite.entity; entry.material=entry.sprite.material; entry.height=entry.sprite.height;
      entry.entity.enabled=this.enabled; entry.sprite.setPaused?.(this.paused);
      if (position) place(entry,position);
      entry.sprite.faceCamera(this.camera);
      return;
    }
    entry.sprite.setClips?.(token);
    if (newSeq && ["idle","walk","attack","hit","fall"].includes(String(token.anim))) entry.sprite.play?.(token.anim);
  }
  snap(entry) {
    place(entry,cellPosition(this.grid,entry.token.cell));
    entry.queue=[]; entry.elapsed=0; entry.start=entry.position.slice();
  }
  advance(entry,dt) {
    if (!entry.queue.length || dt <= 0) return false;
    const pace = paceSeconds(entry.token);
    entry.elapsed+=dt;
    let moved = false;
    while (entry.queue.length && entry.elapsed + 1e-9 >= pace) {
      place(entry,entry.queue.shift()); entry.start=entry.position.slice(); entry.elapsed=Math.max(0,entry.elapsed-pace); moved = true;
    }
    if (!entry.queue.length) {
      entry.elapsed=0;
      if (entry.token.anim === "walk") {
        entry.token={...entry.token,anim:"idle"};
        entry.sprite.play?.("idle");
      }
      return moved;
    }
    const t=entry.elapsed/pace, end=entry.queue[0];
    const position=entry.start.map((value,i)=>value+(end[i]-value)*t);
    const size=this.grid.cell_m ?? 1.524, origin=this.grid.origin ?? [0,0];
    position[1]=terrainHeight(this.grid,(position[0]-origin[0])/size,(position[2]-origin[1])/size);
    place(entry,position);
    return true;
  }
  cameraMoved() {
    const next = cameraXY(this.camera);
    if (!next) return false;
    const previous = this.lastCameraXY;
    this.lastCameraXY = next;
    return !previous || previous[0] !== next[0] || previous[1] !== next[1];
  }
  update(dt) {
    if (this.destroyed) return;
    for (const entry of this.entries.values()) entry.sprite.tick?.();
    if (this.paused) {
      if (this.cameraMoved()) for (const entry of this.entries.values()) entry.sprite.faceCamera(this.camera);
      return;
    }
    const delta=Number.isFinite(dt) ? Math.min(1,Math.max(0,dt)) : 0;
    const moved = [];
    for (const entry of this.entries.values()) if (this.advance(entry,delta)) moved.push(entry);
    const followed=this.entries.get(this.followId);
    if (followed && this.enabled) this.effects?.track([followed.position[0],followed.position[1]+followed.height/2,followed.position[2]],delta);
    const cameraMoved = this.cameraMoved();
    if (cameraMoved) for (const entry of this.entries.values()) entry.sprite.faceCamera(this.camera);
    else for (const entry of moved) entry.sprite.faceCamera(this.camera);
    if (moved.length) this.refresh();
  }
  getState() {
    const tokens=[...this.entries].map(([id,entry])=>({id,name:entry.token.name ?? id,kind:role(entry.token),
      cell:Object.values(positionCell(this.grid,entry.position)),position:entry.position.slice(),moving:entry.queue.length>0}));
    return {count:tokens.length,moving:tokens.filter(token=>token.moving).length,followId:this.followId,tokens};
  }
  refresh() {
    const state=this.getState();
    const occupied = JSON.stringify(state.tokens.map(token => [token.id,token.kind,token.cell[0],token.cell[1]]));
    if (occupied !== this.occupiedSignature) { this.occupiedSignature=occupied; this.cells.update(state.tokens); }
    this.onState(state);
  }
  follow(id) {
    if (this.destroyed || !this.enabled || !this.entries.has(id)) { this.clearFollow(); return false; }
    this.followId=id; this.onState(this.getState()); return true;
  }
  clearFollow() { this.followId=null; this.effects?.stop(); }
  pause(on) { this.paused=Boolean(on); for (const entry of this.entries.values()) entry.sprite.setPaused?.(this.paused); }
  setReducedMotion(on) { this.reducedMotion=Boolean(on); if (on) { for (const entry of this.entries.values()) { this.snap(entry); entry.sprite.faceCamera(this.camera); } this.refresh(); } }
  setEnabled(on) {
    this.enabled=Boolean(on);
    for (const entry of this.entries.values()) entry.entity.enabled=this.enabled;
    this.cells.setEnabled(this.enabled && this.gridVisible);
    if (!this.enabled) this.clearFollow();
  }
  setGridVisible(on) { this.gridVisible=Boolean(on); this.cells.setEnabled(this.enabled && this.gridVisible); }
  destroy() {
    if (this.destroyed) return;
    this.destroyed=true;
    for (const id of this.entries.keys()) this.remove(id);
    this.cells.destroy();
  }
}
/** createTokenController animates authoritative token snapshots and lights their current cells. */
export function createTokenController(options={}) { return new TokenController(options); }
