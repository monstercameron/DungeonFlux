import { createTokenController } from "./token.mjs";

/** applyCamera accepts ordered camera commands and optional token tracking. */
export function applyCamera(state, command) {
  if (!command || !state.camera) return;
  const seq=Number(command.seq ?? 0), preset=command.preset;
  if (!Number.isSafeInteger(seq) || seq<0 || (seq>0 && seq<=state.cameraSequence)) return;
  if (!preset && command.follow === undefined && !command.focus_token_id) return;
  if (seq===0 && preset===state.cameraPreset && command.follow===undefined && !command.focus_token_id) return;
  if (seq>0) state.cameraSequence=seq;
  if (preset && Object.hasOwn(state.cameras,preset)) {
    state.tokens?.clearFollow();
    state.effects?.camera(command);
    state.cameraPreset=preset;
  }
  if (command.follow === false) state.tokens?.clearFollow();
  if (command.follow === true) state.tokens?.follow(command.focus_token_id);
  else if (command.focus_token_id) {
    const token=state.tokens?.getState().tokens.find(value=>value.id===command.focus_token_id);
    if (token) { state.tokens.clearFollow(); state.effects?.track(token.position,1); }
  }
}
/** acceptScene buffers snapshots during loading and orders token updates before focus commands. */
export function acceptScene(state,scene) {
  const seq=Number(scene.seq ?? 0);
  if (!Number.isSafeInteger(seq) || seq<=state.sceneSequence) return;
  state.sceneSequence=seq; state.pendingScene=scene;
  if (state.tokens) { state.tokens.apply(scene); applyCamera(state,scene.camera); }
  state.canvas.style.opacity=scene.visible ? "1" : "0";
  state.canvas.style.transition=scene.visible ? "opacity 600ms ease" : "none";
}
/** attachTokens installs the authoritative token renderer after the filtered grid is ready. */
export function attachTokens(pc,state,grid,layer,message) {
  if (!grid) return;
  state.tokens=createTokenController({pc,app:state.app,grid,layer,camera:state.camera,effects:state.effects,
    reducedMotion:state.reducedMotion,
    onState:value=>{state.tokenState=value;}});
  state.tokens.pause(state.paused);
  const pending=state.pendingScene;
  state.tokens.apply({seq:pending?.seq ?? 0,tokens:pending?.tokens ?? message.tokens ?? []});
  if (pending) applyCamera(state,pending.camera);
  state.app.on("destroy",()=>state.tokens?.destroy());
}
