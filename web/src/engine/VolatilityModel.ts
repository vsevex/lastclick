export class VolatilityModel {
  private margin: number;
  private vol: number;
  private drift: number;
  private elapsedMs = 0;
  private tier: number;

  constructor(tier: number) {
    this.tier = tier;
    this.margin = 0.15 + Math.random() * 0.1;
    this.vol = 0.008 * tier;
    this.drift = 0.0003 * tier;
  }

  tick(dtMs: number): { marginRatio: number; volatilityMul: number } {
    this.elapsedMs += dtMs;
    const timeFactor = 1 + this.elapsedMs / 60000;

    const noise = (Math.random() - 0.48) * this.vol * timeFactor;
    const spike = Math.random() < 0.005 ? Math.random() * 0.05 * this.tier : 0;

    this.margin = Math.max(
      0.01,
      Math.min(1.0, this.margin + this.drift * (dtMs / 100) + noise + spike),
    );

    const volatilityMul =
      timeFactor * (1 + Math.sin(this.elapsedMs / 10000) * 0.15);

    return {
      marginRatio: this.margin,
      volatilityMul: Math.max(0.5, volatilityMul),
    };
  }

  forceMargin(value: number) {
    this.margin = Math.max(0, Math.min(1.0, value));
  }
}
