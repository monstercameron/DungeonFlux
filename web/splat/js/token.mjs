import { adjacentCells, cellPosition, normalizeTokenPath, positionCell, terrainHeight, tokenCell, validTokenCell } from "./occupied_cells.mjs";
import { createOccupiedCells } from "./token_cells.mjs";

const COLORS = { player: [.04,.86,1], villain: [1,.09,.16] };
function role(token) { return ["villain","enemy","monster","thrall"].includes(String(token.kind).toLowerCase()) ? "villain" : "player"; }
function same(a,b) { return a.c === b.c && a.r === b.r; }
function gone(token) { return [token.status,...(Array.isArray(token.statuses) ? token.statuses : [])].some(value => ["removed","fled","defeated"].includes(String(value).toLowerCase())); }
function marker(pc, app, token, layer) {
  const entity = new pc.Entity(`df-token-${token.id}`), kind = role(token);
  const height = Math.max(.6,Math.min(3,Number(token.height_m) || 1.8));
  entity.addComponent("render", { type: kind === "villain" ? "cone" : "capsule", layers: layer == null ? undefined : [layer] });
  entity.setLocalScale?.(.65,kind === "player" ? height/2 : height,.65);
  const material = new pc.StandardMaterial();
  material.diffuse = new pc.Color(...COLORS[kind]);
  material.emissive = new pc.Color(...COLORS[kind]);
  material.useLighting = false;
  material.update();
  entity.render.material = material;
  app.root.addChild(entity);
  return { entity,material,height };
}
function place(entry,position) {
  entry.position = position.slice();
  entry.entity.setPosition(position[0],position[1]+entry.height/2,position[2]);
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
  constructor({pc,app,grid,layer,effects,reducedMotion=false,onState=()=>{}}) {
    Object.assign(this,{pc,app,grid,layer,effects,reducedMotion,onState});
    this.entries = new Map(); this.sceneSeq = -1; this.followId = null;
    this.enabled = true; this.gridVisible = true; this.paused = false; this.destroyed = false;
    this.cells = createOccupiedCells({pc,app,grid,layer});
  }
  remove(id) {
    const entry = this.entries.get(id);
    if (!entry) return;
    entry.entity.destroy(); entry.material.destroy(); this.entries.delete(id);
    if (this.followId === id) this.clearFollow();
  }
  acceptToken(token) {
    let entry = this.entries.get(token.id);
    if (gone(token)) { this.remove(token.id); return; }
    const seq = Number(token.anim_seq ?? 0);
    if (!Number.isSafeInteger(seq) || seq < 0 || !validTokenCell(this.grid,token.cell)) return;
    if (entry && seq <= entry.seq) { entry.token = {...entry.token,name:token.name ?? entry.token.name}; return; }
    if (entry && role(entry.token) !== role(token)) { this.remove(token.id); entry=null; }
    const path = normalizeTokenPath(this.grid,token);
    if (!path) return;
    const signature = JSON.stringify([tokenCell(token.cell),path]);
    if (entry?.signature === signature) { entry.seq=seq; entry.token=token; return; }
    const route = entry ? routeFor(this.grid,entry,token,path) : [];
    if (route === null) return;
    if (!entry) {
      entry = {...marker(this.pc,this.app,token,this.layer), token,seq,queue:[],elapsed:0};
      this.entries.set(token.id,entry);
    }
    entry.token=token; entry.seq=seq; entry.signature=signature;
    entry.queue=route; entry.elapsed=0; entry.start=entry.position?.slice();
    entry.entity.enabled=this.enabled;
    if (!route.length || this.reducedMotion) this.snap(entry);
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
  snap(entry) {
    place(entry,cellPosition(this.grid,entry.token.cell));
    entry.queue=[]; entry.elapsed=0; entry.start=entry.position.slice();
  }
  advance(entry,dt) {
    if (!entry.queue.length) return;
    entry.elapsed+=dt;
    while (entry.queue.length && entry.elapsed + 1e-9 >= .25) {
      place(entry,entry.queue.shift()); entry.start=entry.position.slice(); entry.elapsed=Math.max(0,entry.elapsed-.25);
    }
    if (!entry.queue.length) { entry.elapsed=0; return; }
    const t=entry.elapsed/.25, end=entry.queue[0];
    const position=entry.start.map((value,i)=>value+(end[i]-value)*t);
    const size=this.grid.cell_m ?? 1.524, origin=this.grid.origin ?? [0,0];
    position[1]=terrainHeight(this.grid,(position[0]-origin[0])/size,(position[2]-origin[1])/size);
    place(entry,position);
  }
  update(dt) {
    if (this.destroyed || this.paused) return;
    const delta=Number.isFinite(dt) ? Math.min(1,Math.max(0,dt)) : 0;
    for (const entry of this.entries.values()) this.advance(entry,delta);
    const followed=this.entries.get(this.followId);
    if (followed && this.enabled) this.effects?.track([followed.position[0],followed.position[1]+followed.height/2,followed.position[2]],delta);
    this.refresh();
  }
  getState() {
    const tokens=[...this.entries].map(([id,entry])=>({id,name:entry.token.name ?? id,kind:role(entry.token),
      cell:Object.values(positionCell(this.grid,entry.position)),position:entry.position.slice(),moving:entry.queue.length>0}));
    return {count:tokens.length,moving:tokens.filter(token=>token.moving).length,followId:this.followId,tokens};
  }
  refresh() {
    const state=this.getState();
    this.cells.update(state.tokens); this.onState(state);
  }
  follow(id) {
    if (this.destroyed || !this.enabled || !this.entries.has(id)) { this.clearFollow(); return false; }
    this.followId=id; this.onState(this.getState()); return true;
  }
  clearFollow() { this.followId=null; this.effects?.stop(); }
  pause(on) { this.paused=Boolean(on); }
  setReducedMotion(on) { this.reducedMotion=Boolean(on); if (on) { for (const entry of this.entries.values()) this.snap(entry); this.refresh(); } }
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
