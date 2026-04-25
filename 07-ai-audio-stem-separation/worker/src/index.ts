import { route, warmSpace } from "./handlers.mjs";

export default {
  fetch(request: Request, env: Record<string, string>) {
    return route(request, env);
  },
  async scheduled(_event: ScheduledEvent, env: Record<string, string>, _ctx: ExecutionContext) {
    await warmSpace(env);
  },
};

