const { cyan, dim, magenta, yellow, green } = require('colorette');

// 50 funny security meme strings for vulnerability hunting
const SECURITY_MEMES = [
  "Hunting bugs like a digital bounty hunter...",
  "Scanning for vulnerabilities with my cyber magnifying glass...",
  "Looking for security holes like Swiss cheese inspector...",
  "Channeling my inner hacker detective...",
  "Searching for bugs with the intensity of a caffeinated pentester...",
  "Unleashing the kraken of vulnerability scanners...",
  "Going full Sherlock Holmes on this codebase...",
  "Sniffing out security flaws like a bloodhound...",
  "Diving deep into the rabbit hole of code review...",
  "Activating security spider-sense...",
  "Putting on my white hat and cape...",
  "Scanning with the power of a thousand security tools...",
  "Looking for needles in the haystack of code...",
  "Channeling my inner security ninja...",
  "Going full CSI: Cyber on this project...",
  "Hunting vulnerabilities like Pokemon, gotta catch 'em all...",
  "Scanning code like a security X-ray machine...",
  "Looking for bugs with laser focus...",
  "Unleashing the security audit beast...",
  "Going full Matrix mode on vulnerability detection...",
  "Scanning with the precision of a Swiss watchmaker...",
  "Looking for security gaps like a digital archaeologist...",
  "Channeling my inner cyber warrior...",
  "Hunting bugs with the patience of a zen master...",
  "Scanning code like a security MRI machine...",
  "Looking for vulnerabilities with eagle eyes...",
  "Going full Iron Man with security scanning tech...",
  "Hunting flaws like a digital treasure hunter...",
  "Scanning with the thoroughness of a perfectionist...",
  "Looking for bugs like Where's Waldo, security edition...",
  "Channeling my inner security wizard...",
  "Going full Batman on code vulnerability analysis...",
  "Scanning with the intensity of a security storm...",
  "Looking for holes like a digital cheese inspector...",
  "Hunting vulnerabilities with sniper precision...",
  "Scanning code like a security CT scan...",
  "Looking for flaws with microscopic attention...",
  "Going full Avengers mode on security threats...",
  "Hunting bugs like a digital exterminator...",
  "Scanning with the power of security superpowers...",
  "Looking for vulnerabilities like a code whisperer...",
  "Channeling my inner security Jedi...",
  "Going full Mission Impossible on code infiltration...",
  "Scanning with the dedication of a security monk...",
  "Looking for bugs like a digital fortune teller...",
  "Hunting vulnerabilities with the speed of light...",
  "Scanning code like a security time traveler...",
  "Looking for flaws with the wisdom of security sages...",
  "Going full Terminator on vulnerability elimination..."
];

/**
 * Animated loader with sweeping color effect and spinner
 * Similar to Claude's code animation
 */
class AnimatedLoader {
  constructor(text = 'Loading...', options = {}) {
    this.text = text;
    this.options = {
      spinnerFrames: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'],
      sweepSpeed: 60, // milliseconds between sweep updates (even faster)
      spinnerSpeed: 30, // milliseconds between spinner frames (much faster)
      primaryColor: yellow, // Use orange/yellow as primary color
      dimColor: dim,
      ...options
    };
    
    this.spinnerIndex = 0;
    this.sweepPosition = 0;
    this.isRunning = false;
    this.interval = null;
    this.startTime = null;
  }

  /**
   * Apply sweeping color effect to text
   */
  _applySweepEffect(text, position) {
    const chars = text.split('');
    const coloredChars = chars.map((char, index) => {
      // Calculate distance from sweep position
      const distance = Math.abs(index - position);
      const maxDistance = 5; // Expanded color effect range
      
      if (distance === 0) {
        // Brightest color at sweep position
        return this.options.primaryColor(char);
      } else if (distance === 1) {
        // Very bright - full primary color
        return this.options.primaryColor(char);
      } else if (distance === 2) {
        // Bright - full primary color
        return this.options.primaryColor(char);
      } else if (distance === 3) {
        // Medium brightness - slightly dimmed primary color
        return this.options.dimColor(this.options.primaryColor(char));
      } else if (distance === 4) {
        // More dimmed
        return this.options.dimColor(this.options.primaryColor(char));
      } else if (distance === 5) {
        // Very dimmed
        return this.options.dimColor(this.options.dimColor(char));
      } else {
        // Default dim color
        return this.options.dimColor(char);
      }
    });
    
    return coloredChars.join('');
  }

  /**
   * Get current spinner frame
   */
  _getSpinnerFrame() {
    const frame = this.options.spinnerFrames[this.spinnerIndex];
    this.spinnerIndex = (this.spinnerIndex + 1) % this.options.spinnerFrames.length;
    return this.options.primaryColor(frame);
  }

  /**
   * Update the animation frame
   */
  _updateFrame() {
    if (!this.isRunning) {
      return;
    }

    // Clear the current line
    process.stdout.write('\r\x1b[K');
    
    // Get spinner frame
    const spinner = this._getSpinnerFrame();
    
    // Apply sweep effect to text
    const sweptText = this._applySweepEffect(this.text, this.sweepPosition);
    
    // Calculate elapsed time
    const elapsed = this.startTime ? Date.now() - this.startTime : 0;
    const seconds = Math.floor(elapsed / 1000);
    const timeDisplay = seconds > 0 ? dim(` (${seconds}s)`) : '';
    
    // Write the animated line
    process.stdout.write(`${spinner} ${sweptText}${timeDisplay}`);
    
    // Update sweep position
    this.sweepPosition = (this.sweepPosition + 1) % (this.text.length + 6);
  }

  /**
   * Start the animation
   */
  start() {
    if (this.isRunning) {
      return this;
    }
    
    this.isRunning = true;
    this.startTime = Date.now();
    this.sweepPosition = 0;
    this.spinnerIndex = 0;
    
    // Initial frame
    this._updateFrame();
    
    // Start animation interval
    this.interval = setInterval(() => {
      this._updateFrame();
    }, this.options.spinnerSpeed);
    
    return this;
  }

  /**
   * Stop the animation
   */
  stop(finalMessage = null) {
    if (!this.isRunning) {
      return this;
    }
    
    this.isRunning = false;
    
    if (this.interval) {
      clearInterval(this.interval);
      this.interval = null;
    }
    
    // Clear the current line
    process.stdout.write('\r\x1b[K');
    
    // Show final message if provided
    if (finalMessage) {
      console.log(finalMessage);
    }
    
    return this;
  }

  /**
   * Update the text while animation is running
   */
  updateText(newText) {
    this.text = newText;
    this.sweepPosition = 0; // Reset sweep position for new text
    return this;
  }

  /**
   * Create a promise that resolves when the loader is stopped
   */
  promise() {
    return new Promise((resolve) => {
      const checkStopped = () => {
        if (!this.isRunning) {
          resolve();
        } else {
          setTimeout(checkStopped, 50);
        }
      };
      checkStopped();
    });
  }
}

/**
 * Get a random security meme string
 */
function getRandomSecurityMeme() {
  const randomIndex = Math.floor(Math.random() * SECURITY_MEMES.length);
  return SECURITY_MEMES[randomIndex];
}

/**
 * Convenience function to create and start a loader
 */
function createLoader(text, options) {
  return new AnimatedLoader(text, options).start();
}

/**
 * Convenience function for async operations with loader
 */
async function withLoader(text, asyncFn, options = {}) {
  const loader = createLoader(text, options);
  
  try {
    const result = await asyncFn();
    loader.stop(options.successMessage || `✅ ${text.replace(/\.\.\.$/, '')} completed`);
    return result;
  } catch (error) {
    loader.stop(options.errorMessage || `❌ ${text.replace(/\.\.\.$/, '')} failed`);
    throw error;
  }
}

/**
 * Convenience function for security analysis with random meme text
 */
async function withSecurityLoader(asyncFn, options = {}) {
  // Add a newline before starting the loader for better spacing
  console.log('');
  const randomMeme = getRandomSecurityMeme();
  return await withLoader(randomMeme, asyncFn, options);
}

module.exports = {
  AnimatedLoader,
  createLoader,
  withLoader,
  withSecurityLoader,
  getRandomSecurityMeme
};
