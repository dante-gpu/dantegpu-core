import { gsap } from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { TextPlugin } from 'gsap/TextPlugin';

// Register GSAP plugins
gsap.registerPlugin(ScrollTrigger, TextPlugin);

let animationsInitialized = false;

// SplitText alternative - simple text splitting utility
class SimpleSplitText {
  element: HTMLElement;
  chars: HTMLElement[] = [];
  words: HTMLElement[] = [];
  lines: HTMLElement[] = [];

  constructor(element: HTMLElement | string, options: { type: string; charsClass?: string }) {
    this.element = typeof element === 'string' ? document.querySelector(element)! : element;
    
    if (!this.element) return;

    const text = this.element.textContent || '';
    this.element.innerHTML = '';

    if (options.type === 'chars') {
      this.splitChars(text, options.charsClass);
    } else if (options.type === 'words') {
      this.splitWords(text);
    } else if (options.type === 'lines') {
      this.splitLines(text);
    }
  }

  private splitChars(text: string, charsClass?: string) {
    const chars = text.split('');
    chars.forEach(char => {
      const span = document.createElement('span');
      span.textContent = char === ' ' ? '\u00A0' : char;
      if (charsClass) span.className = charsClass;
      span.style.display = 'inline-block';
      this.chars.push(span);
      this.element.appendChild(span);
    });
  }

  private splitWords(text: string) {
    const words = text.split(' ');
    words.forEach((word, index) => {
      const span = document.createElement('span');
      span.textContent = word;
      span.style.display = 'inline-block';
      this.words.push(span);
      this.element.appendChild(span);
      
      if (index < words.length - 1) {
        this.element.appendChild(document.createTextNode(' '));
      }
    });
  }

  private splitLines(text: string) {
    const div = document.createElement('div');
    div.textContent = text;
    div.style.display = 'inline-block';
    this.lines.push(div);
    this.element.appendChild(div);
  }

  static create(element: HTMLElement | string, options: { type: string; charsClass?: string }) {
    return new SimpleSplitText(element, options);
  }
}

export const initializeAnimations = () => {
  // Prevent multiple initializations
  if (animationsInitialized) {
    console.log("Animations already initialized, skipping...");
    return;
  }
  
  console.log("Initializing GSAP animations...");
  animationsInitialized = true;

  const colorThemes = {
    original: [
      "#340B05", "#0358F7", "#5092C7", "#E1ECFE", "#FFD400", "#FA3D1D", "#FD02F5", "#FFC0FD"
    ],
    "blue-pink": [
      "#1E3A8A", "#3B82F6", "#A855F7", "#EC4899", "#F472B6", "#F9A8D4", "#FBCFE8", "#FDF2F8"
    ],
    "blue-orange": [
      "#1E40AF", "#3B82F6", "#60A5FA", "#FFFFFF", "#FED7AA", "#FB923C", "#EA580C", "#9A3412"
    ],
    sunset: [
      "#FEF3C7", "#FCD34D", "#F59E0B", "#D97706", "#B45309", "#92400E", "#78350F", "#451A03"
    ],
    purple: [
      "#F3E8FF", "#E9D5FF", "#D8B4FE", "#C084FC", "#A855F7", "#9333EA", "#7C3AED", "#6B21B6"
    ],
    monochrome: [
      "#1A1A1A", "#404040", "#666666", "#999999", "#CCCCCC", "#E5E5E5", "#F5F5F5", "#FFFFFF"
    ]
  };

  const darkThemes = ["monochrome"];
  let currentTheme = "original";
  let blurEnabled = true;
  let soundEnabled = false;

  const audioFiles = {
    whoosh: new Audio("https://assets.codepen.io/7558/whoosh-fx-001.mp3"),
    glitch: new Audio("https://assets.codepen.io/7558/glitch-fx-001.mp3"),
    reverb: new Audio("https://assets.codepen.io/7558/click-reverb-001.mp3")
  };

  Object.values(audioFiles).forEach((sound) => {
    sound.volume = 0.3;
  });

  function playSound(soundName: keyof typeof audioFiles) {
    if (soundEnabled && audioFiles[soundName]) {
      audioFiles[soundName].currentTime = 0;
      audioFiles[soundName].play().catch(() => {});
    }
  }

  function blendColors(color1: string, color2: string, percentage: number) {
    percentage = Math.max(0, Math.min(1, percentage));
    const hexToRgb = (hex: string) => {
      const bigint = Number.parseInt(hex.slice(1), 16);
      return [(bigint >> 16) & 255, (bigint >> 8) & 255, bigint & 255];
    };
    const rgbToHex = (rgb: number[]) =>
      "#" +
      ((1 << 24) + (rgb[0] << 16) + (rgb[1] << 8) + rgb[2])
        .toString(16)
        .slice(1);
    const rgb1 = hexToRgb(color1);
    const rgb2 = hexToRgb(color2);
    const r = Math.round(rgb1[0] * (1 - percentage) + rgb2[0] * percentage);
    const g = Math.round(rgb1[1] * (1 - percentage) + rgb2[1] * percentage);
    const b = Math.round(rgb1[2] * (1 - percentage) + rgb2[2] * percentage);
    return rgbToHex([r, g, b]);
  }

  function updateColors(theme: string) {
    const colors = colorThemes[theme as keyof typeof colorThemes];
    if (!colors) return;

    const isDarkTheme = darkThemes.includes(theme);
    document.documentElement.style.setProperty(
      "--color-bg",
      isDarkTheme ? "#1a1a1a" : "#f5f5f5"
    );
    document.documentElement.style.setProperty(
      "--color-text",
      isDarkTheme ? "#ffffff" : "#333"
    );
    document.documentElement.style.setProperty(
      "--color-text-light",
      isDarkTheme ? "#cccccc" : "#666"
    );
    document.documentElement.style.setProperty(
      "--color-text-lighter",
      isDarkTheme ? "#999999" : "#999"
    );

    document
      .querySelectorAll(".main-title, .wavelength-label")
      .forEach((el) => {
        (el as HTMLElement).style.color = isDarkTheme ? "#FFFFFF" : "#333333";
      });

    const emailLink = document.querySelector(".email-link") as HTMLElement;
    if (emailLink) {
      emailLink.style.setProperty(
        "--bg-color",
        isDarkTheme ? "#FFFFFF" : "#333333"
      );
      emailLink.style.setProperty(
        "--hover-color",
        isDarkTheme ? "#333333" : "#f5f5f5"
      );
    }

    document.documentElement.style.setProperty("--grad-color-1", colors[0]);
    document.documentElement.style.setProperty("--grad-color-2", colors[1]);
    document.documentElement.style.setProperty("--grad-color-3", colors[4]);
    document.documentElement.style.setProperty("--grad-color-4", colors[5]);
    document.documentElement.style.setProperty("--grad-color-5", colors[6]);
    document.documentElement.style.setProperty("--grad-color-6", colors[2]);

    updateGradients(theme);
    updateTitleGradient(colors);
  }

  function updateTitleGradient(colors: string[]) {
    document.documentElement.style.setProperty("--grad-1", colors[0] || "#340B05");
    document.documentElement.style.setProperty("--grad-2", colors[1] || "#0358F7");
    document.documentElement.style.setProperty("--grad-3", colors[4] || "#FFD400");
    document.documentElement.style.setProperty("--grad-4", colors[5] || "#FA3D1D");
    document.documentElement.style.setProperty("--grad-5", colors[6] || "#FD02F5");
  }

  let rainbowAnimationTimeline: gsap.core.Timeline | null = null;

  function createRainbowAnimation() {
    const heroTitle = document.querySelector(".hero-title");
    if (!heroTitle) return;
    
    const chars = heroTitle.querySelectorAll(".char");
    const themeColors = colorThemes[currentTheme as keyof typeof colorThemes] || colorThemes.original;
    const defaultTextColor = getComputedStyle(document.documentElement)
      .getPropertyValue("--color-text")
      .trim();

    const waveLength = 6;
    const fadeLength = 3;
    const totalAnimationRange = chars.length + waveLength + fadeLength;

    if (rainbowAnimationTimeline) {
      rainbowAnimationTimeline.kill();
    }
    gsap.killTweensOf(chars);

    gsap.set(chars, {
      color: defaultTextColor,
      opacity: 1,
      filter: "blur(0px)",
      x: 0
    });

    rainbowAnimationTimeline = gsap.timeline({ repeat: 0, ease: "none" });

    rainbowAnimationTimeline.to(
      { x: 0 },
      {
        x: totalAnimationRange,
        duration: 2.5,
        onUpdate: function () {
          const wavePosition = (this.targets()[0] as any).x;
          chars.forEach((char, index) => {
            const charRelativePosition = wavePosition - index;
            let colorToApply = defaultTextColor;

            if (
              charRelativePosition >= 0 &&
              charRelativePosition < totalAnimationRange
            ) {
              if (charRelativePosition < fadeLength) {
                const progress = charRelativePosition / fadeLength;
                colorToApply = blendColors(
                  defaultTextColor,
                  themeColors[0],
                  progress
                );
              } else if (
                charRelativePosition >= fadeLength &&
                charRelativePosition < fadeLength + waveLength
              ) {
                const waveProgress =
                  (charRelativePosition - fadeLength) / waveLength;
                const colorIndex = Math.floor(
                  waveProgress * themeColors.length
                );
                colorToApply =
                  themeColors[Math.min(themeColors.length - 1, colorIndex)];
              } else if (
                charRelativePosition >= fadeLength + waveLength &&
                charRelativePosition < fadeLength + waveLength + fadeLength
              ) {
                const progress =
                  (charRelativePosition - (fadeLength + waveLength)) /
                  fadeLength;
                colorToApply = blendColors(
                  themeColors[themeColors.length - 1],
                  defaultTextColor,
                  progress
                );
              }
            }
            (char as HTMLElement).style.color = colorToApply;
          });
        }
      }
    );
  }

  function toggleBlur() {
    const svgGroup = document.querySelector('g[clip-path="url(#clip)"]');
    playSound("whoosh");

    if (blurEnabled) {
      svgGroup?.removeAttribute("filter");
      blurEnabled = false;
    } else {
      svgGroup?.setAttribute("filter", "url(#blur)");
      blurEnabled = true;
    }
  }

  function updateGradients(theme: string) {
    const colors = colorThemes[theme as keyof typeof colorThemes];
    for (let i = 0; i < 9; i++) {
      const gradient = document.getElementById(`grad${i}`);
      if (gradient && colors) {
        gradient.querySelectorAll("stop").forEach((stop, idx) => {
          if (colors[idx]) stop.setAttribute("stop-color", colors[idx]);
          else if (colors[colors.length - 1])
            stop.setAttribute("stop-color", colors[colors.length - 1]);
        });
      }
    }
  }

  function generateRandomColor() {
    const hue = Math.floor(Math.random() * 360);
    const saturation = Math.floor(Math.random() * 40) + 60;
    const lightness = Math.floor(Math.random() * 50) + 30;
    return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
  }

  function randomizeColors() {
    playSound("glitch");
    const randomColors = Array.from({ length: 8 }, () => generateRandomColor());

    for (let i = 0; i < 9; i++) {
      const gradient = document.getElementById(`grad${i}`);
      if (gradient && randomColors) {
        gradient.querySelectorAll("stop").forEach((stop, idx) => {
          if (randomColors[idx])
            stop.setAttribute("stop-color", randomColors[idx]);
          else if (randomColors[randomColors.length - 1])
            stop.setAttribute(
              "stop-color",
              randomColors[randomColors.length - 1]
            );
        });
      }
    }

    document.documentElement.style.setProperty("--grad-color-1", randomColors[0]);
    document.documentElement.style.setProperty("--grad-color-2", randomColors[1]);
    document.documentElement.style.setProperty("--grad-color-3", randomColors[4]);
    document.documentElement.style.setProperty("--grad-color-4", randomColors[5]);
    document.documentElement.style.setProperty("--grad-color-5", randomColors[6]);
    document.documentElement.style.setProperty("--grad-color-6", randomColors[2]);

    updateTitleGradient(randomColors);

    const tempTheme = currentTheme;
    currentTheme = "randomized";
    setTimeout(() => {
      createRainbowAnimation();
      currentTheme = tempTheme;
    }, 100);

    gsap.to(".svg-container", {
      scale: 1.02,
      duration: 0.2,
      yoyo: true,
      repeat: 1,
      ease: "power2.inOut"
    });
  }

  function toggleSound() {
    const waveLine = document.querySelector(".wave-line");
    if (soundEnabled) {
      waveLine?.classList.remove("wave-animated");
      soundEnabled = false;
    } else {
      waveLine?.classList.add("wave-animated");
      soundEnabled = true;
    }
  }

  // Event listeners
  document.querySelector(".blur-btn")?.addEventListener("click", toggleBlur);
  document.querySelector(".sound-toggle")?.addEventListener("click", toggleSound);

  const heroNavItems = document.querySelectorAll(".hero-nav-item");
  const heroNav = document.querySelector(".hero-nav");
  const gradientOverlay = document.querySelector(".gradient-overlay") as HTMLElement;
  const navGradients = [
    "radial-gradient(circle, #340B05 0%, #0358F7 50%, transparent 100%)",
    "radial-gradient(circle, #0358F7 0%, #5092C7 50%, transparent 100%)",
    "radial-gradient(circle, #FFD400 0%, #FA3D1D 50%, transparent 100%)",
    "radial-gradient(circle, #5092C7 0%, #E1ECFE 50%, transparent 100%)",
    "radial-gradient(circle, #FA3D1D 0%, #FD02F5 50%, transparent 100%)",
    "radial-gradient(circle, #E1ECFE 0%, #FFD400 50%, transparent 100%)",
    "radial-gradient(circle, #FD02F5 0%, #340B05 50%, transparent 100%)",
    "radial-gradient(circle, #FFD400 0%, #5092C7 50%, transparent 100%)",
    "radial-gradient(circle, #5092C7 0%, #FD02F5 50%, transparent 100%)"
  ];

  if (heroNav) {
    heroNav.addEventListener("mouseleave", () => {
      heroNavItems.forEach((navItem) => {
        (navItem as HTMLElement).style.opacity = "1";
        navItem.classList.remove("active");
      });
      if (gradientOverlay) gradientOverlay.style.opacity = "0";
    });
  }

  heroNavItems.forEach((item, index) => {
    item.addEventListener("mouseenter", () => {
      playSound("reverb");
      if (gradientOverlay) {
        gradientOverlay.style.background = navGradients[index];
        gradientOverlay.style.opacity = "0.3";
      }
      heroNavItems.forEach((navItem, navIndex) => {
        navItem.classList.remove("active");
        const distance = Math.abs(index - navIndex);
        let opacity = 1;
        if (navIndex === index) {
          opacity = 1;
          navItem.classList.add("active");
        } else if (distance === 1) {
          opacity = 0.6;
        } else if (distance === 2) {
          opacity = 0.4;
        } else if (distance === 3) {
          opacity = 0.3;
        } else if (distance >= 4) {
          opacity = 0.2;
        }
        (navItem as HTMLElement).style.opacity = opacity.toString();
      });
    });
  });

  function isMobile() {
    return window.innerWidth <= 768;
  }

  // Initialize animations after fonts are ready
  document.fonts.ready.then(() => {
    updateColors(currentTheme);

    // Only create timeline if elements haven't been animated yet
    const heroTitle = document.querySelector(".hero-title") as HTMLElement;
    if (!heroTitle || heroTitle.dataset.animated === 'true') {
      return;
    }

    const heroTl = gsap.timeline({ 
      delay: 0.5,
      onComplete: () => {
        // Mark elements as animated to prevent re-animation
        document.querySelectorAll('.hero-title, .hero-nav-item, .hero-text, .nav-bottom-center').forEach(el => {
          (el as HTMLElement).dataset.animated = 'true';
        });
      }
    });

    // Title animation
    let titleSplit: SimpleSplitText;
    let titleElementsToAnimate: HTMLElement[];
    let titleStagger: number;

    if (isMobile()) {
      titleSplit = SimpleSplitText.create(heroTitle, { type: "words" });
      titleElementsToAnimate = titleSplit.words;
      titleStagger = 0.1;
    } else {
      titleSplit = SimpleSplitText.create(heroTitle, {
        type: "chars",
        charsClass: "char"
      });
      titleElementsToAnimate = titleSplit.chars;
      titleStagger = 0.03;
    }

    gsap.set(titleElementsToAnimate, {
      opacity: 0,
      filter: "blur(8px)",
      x: -20
    });
    heroTl.to(
      titleElementsToAnimate,
      {
        opacity: 1,
        filter: "blur(0px)",
        x: 0,
        duration: 0.8,
        stagger: titleStagger,
        ease: "power2.out"
      },
      0
    );

    // Nav items animation
    const navItems = document.querySelectorAll(".hero-nav-item");
    navItems.forEach((item) => {
      if ((item as HTMLElement).dataset.animated === 'true') return;
      
      const split = SimpleSplitText.create(item as HTMLElement, { type: "lines" });
      gsap.set(split.lines, { opacity: 0, y: 30, filter: "blur(8px)" });
      heroTl.to(
        split.lines,
        {
          autoAlpha: 1,
          opacity: 1,
          y: 0,
          filter: "blur(0px)",
          duration: 0.8,
          stagger: 0.08,
          ease: "power2.out"
        },
        0.4
      );
    });

    // Text content animation
    const textElements = document.querySelectorAll(".hero-text");
    textElements.forEach((textEl, index) => {
      if ((textEl as HTMLElement).dataset.animated === 'true') return;
      
      const textSplit = SimpleSplitText.create(textEl as HTMLElement, { type: "lines" });
      gsap.set(textSplit.lines, {
        opacity: 0,
        y: 50,
        clipPath: "inset(0 0 100% 0)"
      });
      heroTl.to(
        textSplit.lines,
        {
          opacity: 1,
          y: 0,
          clipPath: "inset(0 0 0% 0)",
          duration: 0.8,
          stagger: 0.1,
          ease: "power2.out"
        },
        0.8 + index * 0.2
      );
    });

    // Scroll hint animation
    const scrollHint = document.querySelector(".nav-bottom-center") as HTMLElement;
    if (scrollHint && scrollHint.dataset.animated !== 'true') {
      const scrollHintSplit = SimpleSplitText.create(scrollHint, { type: "chars" });
      gsap.set(scrollHintSplit.chars, { opacity: 0, filter: "blur(3px)" });
      gsap.to(scrollHintSplit.chars, {
        opacity: 1,
        filter: "blur(0px)",
        duration: 0.6,
        stagger: { each: 0.08, repeat: -1, yoyo: true },
        ease: "sine.inOut",
        delay: 1
      });
    }

    // Scroll-triggered animations
    // Only create scroll trigger if not already created
    if (!ScrollTrigger.getById("main-scroll-trigger")) {
      const tl = gsap.timeline({
        scrollTrigger: {
          id: "main-scroll-trigger",
          trigger: ".animation-section",
          start: "top bottom",
          end: "bottom bottom",
          scrub: 1,
        }
      });

      const wavelengthLabels = document.querySelectorAll(".wavelength-label");
      const mainTitle = document.querySelector(".main-title") as HTMLElement;

      const allSplitLines: HTMLElement[] = [];
      wavelengthLabels.forEach((label) => {
        const split = SimpleSplitText.create(label as HTMLElement, { type: "lines" });
        gsap.set(split.lines, { opacity: 0, y: 30, filter: "blur(8px)" });
        allSplitLines.push(...split.lines);
      });
      
      if (mainTitle) {
        const mainTitleSplit = SimpleSplitText.create(mainTitle, { type: "lines" });
        gsap.set(mainTitleSplit.lines, { opacity: 0, y: 30, filter: "blur(8px)" });
        allSplitLines.push(...mainTitleSplit.lines);
      }

      tl.to(".svg-container", { autoAlpha: 1, duration: 0.01 }, 0)
        .to(".text-grid", { autoAlpha: 1, duration: 0.01 }, 0)
        .to(".main-title", { autoAlpha: 1, duration: 0.01 }, 0)
        .to(
          ".svg-container",
          {
            transform: "scaleY(0.05) translateY(-30px)",
            duration: 0.3,
            ease: "power2.out"
          },
          0
        )
        .to(
          ".svg-container",
          {
            transform: "scaleY(1) translateY(0px)",
            duration: 1.2,
            ease: "power2.out"
          },
          0.3
        )
        .to(
          ".nav-bottom-left, .nav-bottom-right, .nav-bottom-center",
          {
            opacity: 0,
            duration: 0.6,
            ease: "power2.out"
          },
          0.2
        )
        .to(
          allSplitLines,
          {
            duration: 0.8,
            y: 0,
            opacity: 1,
            filter: "blur(0px)",
            stagger: 0.08,
            ease: "power2.out"
          },
          0.9
        )
        .to(".level-5", { y: "-25vh", duration: 0.8, ease: "power2.out" }, 0.9)
        .to(".level-4", { y: "-20vh", duration: 0.8, ease: "power2.out" }, 0.9)
        .to(".level-3", { y: "-15vh", duration: 0.8, ease: "power2.out" }, 0.9)
        .to(".level-2", { y: "-10vh", duration: 0.8, ease: "power2.out" }, 0.9)
        .to(".level-1", { y: "-5vh", duration: 0.8, ease: "power2.out" }, 0.9);
    }
  });

  // Add resize listener only once
  if (!window.scrollTriggerResizeAdded) {
    window.addEventListener("resize", () => ScrollTrigger.refresh());
    (window as any).scrollTriggerResizeAdded = true;
  }
};