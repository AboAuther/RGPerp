import { useEffect, useRef } from 'react'

type Pointer = {
  x: number
  y: number
}

type PrismBackgroundProps = {
  pointer: Pointer
}

const vertexShaderSource = `
attribute vec2 a_position;
void main() {
  gl_Position = vec4(a_position, 0.0, 1.0);
}
`

const fragmentShaderSource = `
precision highp float;

uniform vec2 u_resolution;
uniform float u_time;
uniform vec2 u_pointer;

float sdSegment(vec2 p, vec2 a, vec2 b) {
  vec2 pa = p - a;
  vec2 ba = b - a;
  float h = clamp(dot(pa, ba) / dot(ba, ba), 0.0, 1.0);
  return length(pa - ba * h);
}

float glowLine(vec2 p, vec2 a, vec2 b, float width, float intensity) {
  float d = sdSegment(p, a, b);
  return pow(width / max(d, 0.0001), intensity);
}

float triangleMask(vec2 p, vec2 a, vec2 b, vec2 c) {
  vec2 v0 = c - a;
  vec2 v1 = b - a;
  vec2 v2 = p - a;
  float dot00 = dot(v0, v0);
  float dot01 = dot(v0, v1);
  float dot02 = dot(v0, v2);
  float dot11 = dot(v1, v1);
  float dot12 = dot(v1, v2);
  float invDenom = 1.0 / max(dot00 * dot11 - dot01 * dot01, 0.0001);
  float u = (dot11 * dot02 - dot01 * dot12) * invDenom;
  float v = (dot00 * dot12 - dot01 * dot02) * invDenom;
  return step(0.0, u) * step(0.0, v) * step(u + v, 1.0);
}

mat2 rotation(float angle) {
  float s = sin(angle);
  float c = cos(angle);
  return mat2(c, -s, s, c);
}

float hash(vec2 p) {
  return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453123);
}

void main() {
  vec2 uv = (gl_FragCoord.xy - 0.5 * u_resolution.xy) / min(u_resolution.x, u_resolution.y);
  vec2 pointer = (u_pointer - 0.5 * u_resolution.xy) / min(u_resolution.x, u_resolution.y);

  uv *= rotation(pointer.x * 0.22);
  uv.y += 0.03 * sin(u_time * 0.3);

  vec2 apex = vec2(0.0, -0.60);
  vec2 leftBase = vec2(-0.80, 0.62);
  vec2 rightBase = vec2(0.80, 0.62);

  float leftEdge = glowLine(uv, apex, leftBase, 0.010, 1.22);
  float rightEdge = glowLine(uv, apex, rightBase, 0.010, 1.22);
  float baseEdge = glowLine(uv, leftBase, rightBase, 0.014, 1.12);
  float tri = triangleMask(uv, apex, leftBase, rightBase);

  float beamNoise = 0.68 + 0.32 * sin(u_time * 0.7 + uv.y * 11.0 + uv.x * 4.0);
  float beamNoise2 = 0.68 + 0.32 * sin(u_time * 0.9 - uv.y * 9.0 + uv.x * 3.0);

  vec3 color = vec3(0.008, 0.014, 0.028);
  color += vec3(0.010, 0.020, 0.038) * exp(-3.2 * length(uv));

  vec3 leftColor = vec3(0.65, 0.95, 1.0);
  vec3 rightColor = vec3(0.58, 0.82, 1.0);
  vec3 floorColor = vec3(0.82, 0.96, 1.0);

  float leftBeamMask = tri * smoothstep(0.72, 0.05, abs(uv.x + 0.19));
  float rightBeamMask = tri * smoothstep(0.72, 0.05, abs(uv.x - 0.19));

  color += leftColor * leftBeamMask * beamNoise * 0.28;
  color += rightColor * rightBeamMask * beamNoise2 * 0.24;

  color += leftColor * leftEdge * 0.014;
  color += rightColor * rightEdge * 0.014;
  color += floorColor * baseEdge * 0.016;

  float floorGlow = exp(-18.0 * abs(uv.y - 0.63)) * smoothstep(1.05, 0.15, abs(uv.x));
  color += floorColor * floorGlow * 0.72;

  float apexGlow = exp(-20.0 * distance(uv, apex + vec2(pointer.x * 0.18, pointer.y * 0.12)));
  color += vec3(1.0, 0.94, 0.78) * apexGlow * 0.24;

  float centerBloom = exp(-9.0 * length(uv * vec2(0.82, 1.05)));
  color += vec3(0.18, 0.32, 0.62) * centerBloom * 0.12;

  float grain = hash(gl_FragCoord.xy + u_time) * 0.035;
  color += grain;

  float vignette = smoothstep(1.35, 0.2, length(uv));
  color *= vignette;

  gl_FragColor = vec4(color, 1.0);
}
`

function compileShader(gl: WebGLRenderingContext, type: number, source: string) {
  const shader = gl.createShader(type)
  if (!shader) {
    throw new Error('failed to create shader')
  }

  gl.shaderSource(shader, source)
  gl.compileShader(shader)

  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    const log = gl.getShaderInfoLog(shader) ?? 'unknown shader compile error'
    gl.deleteShader(shader)
    throw new Error(log)
  }

  return shader
}

export default function PrismBackground({ pointer }: PrismBackgroundProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const pointerRef = useRef(pointer)

  useEffect(() => {
    pointerRef.current = pointer
  }, [pointer])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) {
      return
    }

    const gl = canvas.getContext('webgl', {
      alpha: true,
      antialias: true,
      premultipliedAlpha: true,
    })

    if (!gl) {
      return
    }

    let frameId = 0

    const vertexShader = compileShader(gl, gl.VERTEX_SHADER, vertexShaderSource)
    const fragmentShader = compileShader(gl, gl.FRAGMENT_SHADER, fragmentShaderSource)

    const program = gl.createProgram()
    if (!program) {
      gl.deleteShader(vertexShader)
      gl.deleteShader(fragmentShader)
      return
    }

    gl.attachShader(program, vertexShader)
    gl.attachShader(program, fragmentShader)
    gl.linkProgram(program)

    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
      gl.deleteProgram(program)
      gl.deleteShader(vertexShader)
      gl.deleteShader(fragmentShader)
      return
    }

    const positionBuffer = gl.createBuffer()
    gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer)
    gl.bufferData(
      gl.ARRAY_BUFFER,
      new Float32Array([
        -1, -1,
        1, -1,
        -1, 1,
        -1, 1,
        1, -1,
        1, 1,
      ]),
      gl.STATIC_DRAW,
    )

    const positionLocation = gl.getAttribLocation(program, 'a_position')
    const resolutionLocation = gl.getUniformLocation(program, 'u_resolution')
    const timeLocation = gl.getUniformLocation(program, 'u_time')
    const pointerLocation = gl.getUniformLocation(program, 'u_pointer')

    const resize = () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2)
      const width = Math.floor(window.innerWidth * dpr)
      const height = Math.floor(window.innerHeight * dpr)

      if (canvas.width !== width || canvas.height !== height) {
        canvas.width = width
        canvas.height = height
        canvas.style.width = `${window.innerWidth}px`
        canvas.style.height = `${window.innerHeight}px`
      }

      gl.viewport(0, 0, width, height)
    }

    const render = (time: number) => {
      resize()

      gl.clearColor(0, 0, 0, 0)
      gl.clear(gl.COLOR_BUFFER_BIT)
      gl.useProgram(program)

      gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer)
      gl.enableVertexAttribArray(positionLocation)
      gl.vertexAttribPointer(positionLocation, 2, gl.FLOAT, false, 0, 0)

      gl.uniform2f(resolutionLocation, canvas.width, canvas.height)
      gl.uniform1f(timeLocation, time * 0.001)
      gl.uniform2f(pointerLocation, pointerRef.current.x, canvas.height / (window.devicePixelRatio || 1) - pointerRef.current.y)

      gl.drawArrays(gl.TRIANGLES, 0, 6)

      frameId = window.requestAnimationFrame(render)
    }

    resize()
    frameId = window.requestAnimationFrame(render)
    window.addEventListener('resize', resize)

    return () => {
      window.cancelAnimationFrame(frameId)
      window.removeEventListener('resize', resize)
      gl.deleteBuffer(positionBuffer)
      gl.deleteProgram(program)
      gl.deleteShader(vertexShader)
      gl.deleteShader(fragmentShader)
    }
  }, [])

  return <canvas ref={canvasRef} className="landing-prism-canvas" aria-hidden />
}
